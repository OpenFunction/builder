// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implements java/functions_framework buildpack.
// The functions_framework buildpack copies the function framework into a layer, and adds it to a compiled function to make an executable app.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/beevik/etree"
	"github.com/buildpacks/libcnb"
)

const (
	layerName = "functions-framework"

	defaultMavenRepository     = "https://repo.maven.apache.org/maven2/"
	defaultFrameworkGroup      = "dev.openfunction.functions"
	defaultFrameworkArtifactID = "functions-framework-invoker"
	defaultFrameworkVersion    = "1.0.0"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(_ *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	layer := ctx.Layer(layerName)
	layer.Launch = true

	if err := installFunctionsFramework(ctx, layer); err != nil {
		return err
	}

	classpath, err := classpath(ctx)
	if err != nil {
		return err
	}
	layer.LaunchEnvironment.Default(env.FunctionClasspath, classpath)

	ctx.SetFunctionsEnvVars(layer)

	// Check that the classes are indeed in the classpath we just determined.
	if !checkTargets(ctx, classpath) {
		return gcp.UserErrorf("build succeeded but did not produce the target classes")
	}

	launcherSource := filepath.Join(ctx.BuildpackRoot(), "launch.sh")
	launcherTarget := filepath.Join(layer.Path, "launch.sh")
	createLauncher(ctx, launcherSource, launcherTarget)
	ctx.AddDefaultWebProcess([]string{launcherTarget, "java", "-jar", filepath.Join(layer.Path, "functions-framework.jar")}, true)

	return nil
}

// checkTargets use javap to check that the class is indeed in the classpath we just determined.
// On success, it will output a description of the class and its public members, which we discard.
// On failure it will output an error saying what's wrong (usually that the class doesn't exist).
// Success here doesn't guarantee that the function will execute. It might not implement one of the
// required interfaces, for example. But it eliminates the commonest problem of specifying the wrong target.
// We use an ExecUser* method so that the time taken by the javap command is counted as user time.
func checkTargets(ctx *gcp.Context, classpath string) bool {
	targets := strings.Split(os.Getenv(env.FunctionTarget), ",")
	success := true
	for _, target := range targets {
		cmd := []string{"javap", "-classpath", classpath, target}
		if _, err := ctx.ExecWithErr(cmd, gcp.WithUserAttribution); err != nil {
			// The javap error output will typically be "Error: class not found: foo.Bar".
			success = false
		}
	}

	return success
}

func createLauncher(ctx *gcp.Context, launcherSource, launcherTarget string) {
	launcherContents := ctx.ReadFile(launcherSource)
	ctx.WriteFile(launcherTarget, launcherContents, 0755)
}

// classpath determines what the --classpath argument should be. This tells the Functions Framework where to find
// the classes of the function, including dependencies.
func classpath(ctx *gcp.Context) (string, error) {
	if ctx.FileExists("pom.xml") {
		return mavenClasspath(ctx)
	}
	if ctx.FileExists("build.gradle") {
		return gradleClasspath(ctx)
	}
	jars := ctx.Glob("*.jar")
	if len(jars) == 1 {
		// Already-built jar file. It should be self-contained, which means that it can be the only thing given to --classpath.
		return jars[0], nil
	}
	if len(jars) > 1 {
		return "", gcp.UserErrorf("function has no pom.xml and more than one jar file: %s", strings.Join(jars, ", "))
	}
	// We have neither pom.xml nor a jar file. Show what files there are. If the user deployed the wrong directory, this may help them see the problem more easily.
	description := "directory is empty"
	if files := ctx.Glob("*"); len(files) > 0 {
		description = fmt.Sprintf("directory has these entries: %s", strings.Join(files, ", "))
	}
	return "", gcp.UserErrorf("function has neither pom.xml nor already-built jar file; %s", description)
}

// mavenClasspath determines the --classpath when there is a pom.xml. This will consist of the jar file built
// from the pom.xml itself, plus all jar files that are dependencies mentioned in the pom.xml.
func mavenClasspath(ctx *gcp.Context) (string, error) {

	mvn := "mvn"

	// If this project has the Maven Wrapper, we should use it
	if ctx.FileExists("mvnw") {
		mvn = "./mvnw"
	}

	command := []string{mvn, "--batch-mode", "dependency:copy-dependencies"}
	if !ctx.Debug() && !devmode.Enabled(ctx) {
		command = append(command, "--quiet")
	}

	// Copy the dependencies of the function (`<dependencies>` in pom.xml) into target/dependency.
	ctx.Exec(command, gcp.WithUserAttribution)

	// Extract the artifact/version coordinates from the user's pom.xml definitions.
	// mvn help:evaluate is quite slow so we do it this way rather than calling it twice.
	// The name of the built jar file will be <artifact>-<version>.jar, for example myfunction-0.9.jar.
	execResult := ctx.Exec([]string{mvn, "help:evaluate", "-q", "-DforceStdout", "-Dexpression=project.artifactId/${project.version}"}, gcp.WithUserAttribution)
	groupArtifactVersion := execResult.Stdout
	components := strings.Split(groupArtifactVersion, "/")
	if len(components) != 2 {
		return "", gcp.UserErrorf("could not parse query output into artifact/version: %s", groupArtifactVersion)
	}
	artifact, version := components[0], components[1]
	jarName := fmt.Sprintf("target/%s-%s.jar", artifact, version)
	if !ctx.FileExists(jarName) {
		return "", gcp.UserErrorf("expected output jar %s does not exist", jarName)
	}

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return jarName + ":target/dependency/*", nil
}

// gradleClasspath determines the --classpath when there is a build.gradle. This will consist of the jar file built
// from the build.gradle, plus all jar files that are dependencies mentioned there.
// Unlike Maven, Gradle doesn't have a simple way to query the contents of the build.gradle. But we can update
// the user's build.gradle to append tasks that do that. This is a bit ugly, but using --init-script didn't work
// because apparently you can't define tasks there; and having the predefined script include the user's build.gradle
// didn't work very well either, because you can't use a plugins {} clause in an included script.
func gradleClasspath(ctx *gcp.Context) (string, error) {
	extraTasksSource := filepath.Join(ctx.BuildpackRoot(), "extra_tasks.gradle")
	extraTasksText := ctx.ReadFile(extraTasksSource)
	if err := os.Chmod("build.gradle", 0644); err != nil {
		return "", gcp.InternalErrorf("making build.gradle writable: %v", err)
	}
	f, err := os.OpenFile("build.gradle", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return "", gcp.InternalErrorf("opening build.gradle for appending: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := f.Write(extraTasksText); err != nil {
		return "", gcp.InternalErrorf("appending extra definitions to build.gradle: %v", err)
	}

	// Copy the dependencies of the function (`dependencies {...}` in build.gradle) into build/_javaFunctionDependencies.
	ctx.Exec([]string{"gradle", "--quiet", "_javaFunctionCopyAllDependencies"}, gcp.WithUserAttribution)

	// Extract the name of the target jar.
	execResult := ctx.Exec([]string{"gradle", "--quiet", "_javaFunctionPrintJarTarget"}, gcp.WithUserAttribution)
	jarName := strings.TrimSpace(execResult.Stdout)
	if !ctx.FileExists(jarName) {
		return "", gcp.UserErrorf("expected output jar %s does not exist", jarName)
	}

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return fmt.Sprintf("%s:build/_javaFunctionDependencies/*", jarName), nil
}

func installFunctionsFramework(ctx *gcp.Context, layer *libcnb.Layer) error {

	ffName := filepath.Join(layer.Path, "functions-framework.jar")
	if jarPath, ok := os.LookupEnv(env.FunctionFrameworkJar); ok {
		if strings.HasPrefix(jarPath, "http") || strings.HasPrefix(jarPath, "https") {
			return downloadFramework(ctx, ffName, jarPath)
		} else {
			_, err := ctx.ExecWithErr([]string{"bash", "-c", fmt.Sprintf("mv %s %s", jarPath, ffName)})
			return err
		}
	}

	mavenRepository := os.Getenv(env.MavenRepository)
	if mavenRepository == "" {
		mavenRepository = defaultMavenRepository
	}

	frameworkGroup := os.Getenv(env.FunctionFrameworkGroup)
	if frameworkGroup == "" {
		frameworkGroup = defaultFrameworkGroup
	}
	frameworkGroup = strings.ReplaceAll(frameworkGroup, ".", "/")

	frameworkArtifactID := os.Getenv(env.FunctionFrameworkArtifactID)
	if frameworkArtifactID == "" {
		frameworkArtifactID = defaultFrameworkArtifactID
	}

	frameworkVersion := os.Getenv(env.FunctionFrameworkVersion)
	if frameworkVersion == "" {
		frameworkVersion = defaultFrameworkVersion
	}

	artifact := fmt.Sprintf("%s-jar-with-dependencies.jar", frameworkArtifactID)
	version := frameworkVersion
	if strings.HasSuffix(frameworkVersion, "-SNAPSHOT") {
		var err error
		version, err = getSnapshotVersion(ctx, mavenRepository, frameworkGroup, frameworkArtifactID, frameworkVersion)
		if err != nil {
			return err
		}
	}

	artifact = fmt.Sprintf("%s-%s-jar-with-dependencies.jar", frameworkArtifactID, version)
	url := fmt.Sprintf("%s/%s/%s/%s/%s", mavenRepository, frameworkGroup, frameworkArtifactID, frameworkVersion, artifact)

	return downloadFramework(ctx, ffName, strings.ReplaceAll(url, "//", "/"))
}

func downloadFramework(ctx *gcp.Context, name, url string) error {
	_, err := ctx.ExecWithErr([]string{"curl", "--silent", "--fail", "--show-error", "--output", name, url})
	if err != nil {
		return gcp.InternalErrorf("fetching functions framework jar[%s]: %s", url, err.Error())
	}

	ctx.Logf("fetching functions framework jar from %s", url)

	return nil
}

func getSnapshotVersion(ctx *gcp.Context, mavenRepository, frameworkGroup, frameworkArtifactID, frameworkVersion string) (string, error) {

	url := fmt.Sprintf("%s/%s/%s/%s/maven-metadata.xml", mavenRepository, frameworkGroup, frameworkArtifactID, frameworkVersion)
	if _, err := ctx.ExecWithErr([]string{"curl", "--silent", "--fail", "--show-error", "--output", "/tmp/maven-metadata.xml", url}); err != nil {
		return "", gcp.InternalErrorf("fetching functions framework metadata[%s]: %s", url, err.Error())
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromFile("/tmp/maven-metadata.xml"); err != nil {
		return "", gcp.InternalErrorf("parse functions framework metadata[%s]: %s", url, err.Error())
	}

	element := doc.FindElement("//metadata/versioning/snapshotVersions/snapshotVersion[classifier='jar-with-dependencies'][extension='jar']/value")
	if element != nil {
		return element.Text(), nil
	}

	return "", nil
}
