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

// Implements java/gradle buildpack.
// The gradle buildpack builds Gradle applications.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	defaultGradleVersion = "7.4.2"
	gradleDistroURL      = "https://services.gradle.org/distributions/gradle-%s-bin.zip"
	gradlePath           = "/usr/local/gradle"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.FileExists("build.gradle") {
		return gcp.OptInFileFound("build.gradle"), nil
	}
	if ctx.FileExists("build.gradle.kts") {
		return gcp.OptInFileFound("build.gradle.kts"), nil
	}
	return gcp.OptOut("neither build.gradle nor build.gradle.kts found"), nil
}

func buildFn(ctx *gcp.Context) error {

	var gradle string
	if ctx.FileExists("gradlew") {
		gradle = "./gradlew"
	} else {
		if !gradleInstalled() {
			if err := installGradle(ctx); err != nil {
				return fmt.Errorf("installing Gradle: %w", err)
			}
		}

		gradle = "gradle"
	}

	ctx.Exec([]string{gradle, "-v"}, gcp.WithUserAttribution)

	command := []string{gradle, "clean", "assemble", "-x", "test", "--build-cache"}

	if buildArgs := os.Getenv(env.BuildArgs); buildArgs != "" {
		if strings.Contains(buildArgs, "project-cache-dir") {
			ctx.Warnf("Detected project-cache-dir property set in GOOGLE_BUILD_ARGS. Dependency caching may not work properly.")
		}
		command = append(command, buildArgs)
	}

	if !ctx.Debug() && !devmode.Enabled(ctx) {
		command = append(command, "--quiet")
	}

	ctx.Exec(command, gcp.WithUserAttribution)
	return nil
}

func gradleInstalled() bool {
	if version := os.Getenv(env.GradleVersion); version != "" && version != defaultGradleVersion {
		return false
	}
	return true
}

// installGradle installs Gradle and returns the path of the gradle binary
func installGradle(ctx *gcp.Context) error {

	gradleVersion := os.Getenv(env.GradleVersion)
	downloadURL := fmt.Sprintf(gradleDistroURL, gradleVersion)
	// Download and install gradle in layer.
	ctx.Logf("Installing Gradle v%s", gradleVersion)
	if code := ctx.HTTPStatus(downloadURL); code != http.StatusOK {
		return fmt.Errorf("gradle version %s does not exist at %s (status %d)", gradleVersion, downloadURL, code)
	}

	tmpDir := "/tmp"
	gradleZip := filepath.Join(tmpDir, "gradle.zip")
	defer ctx.RemoveAll(gradleZip)

	curl := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s --output %s", downloadURL, gradleZip)
	ctx.Exec([]string{"bash", "-c", curl}, gcp.WithUserAttribution)

	unzip := fmt.Sprintf("unzip -q %s -d %s", gradleZip, tmpDir)
	ctx.Exec([]string{"bash", "-c", unzip}, gcp.WithUserAttribution)

	gradleExtracted := filepath.Join(tmpDir, fmt.Sprintf("gradle-%s", gradleVersion))
	defer ctx.RemoveAll(gradleExtracted)
	install := fmt.Sprintf("mv %s %s", gradleExtracted, gradlePath)
	ctx.Exec([]string{"bash", "-c", install}, gcp.WithUserTimingAttribution)

	command := fmt.Sprintf("rm -rf %s/current", gradlePath)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	command = fmt.Sprintf("ln -s %s/apache-maven-%s %s/current", gradlePath, gradleVersion, gradlePath)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	command = fmt.Sprintf("rm -rf %s/apache-maven-%s", gradlePath, defaultGradleVersion)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	return nil
}
