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

// Implements go/functions_framework buildpack.
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

const (
	layerName                          = "functions-framework"
	gopathLayerName                    = "gopath"
	functionsFrameworkModule           = "github.com/OpenFunction/functions-framework-go"
	functionsFrameworkPackage          = functionsFrameworkModule + "/framework"
	functionsFrameworkFunctionsPackage = functionsFrameworkModule + "/functions"
	appName                            = "serverless_function_app"
	fnSourceDir                        = "serverless_function_source_code"
	customPluginDir                    = "plugins"
)

var (
	tmplV0                    = template.Must(template.New("main").Parse(mainTextTemplate))
	tmplDeclarative           = template.Must(template.New("main_declarative").Parse(mainTextTemplateDeclarative))
	functionsFrameworkVersion = "v0.3.0"
)

type fnInfo struct {
	Source  string
	Target  string
	Package string
	Imports map[string]struct{}
	Plugins []Plugin
}

type parsedPackage struct {
	Name    string              `json:"name"`
	Imports map[string]struct{} `json:"imports"`
}

type Plugin struct {
	AliasName   string
	Path        string
	GetNameFunc string
	NewFunc     string
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer)
	ctx.SetFunctionsEnvVars(l)

	fnTarget := os.Getenv(env.FunctionTarget)

	// Move the function source code into a subdirectory in order to construct the app in the main application root.
	ctx.RemoveAll(fnSourceDir)
	ctx.MkdirAll(fnSourceDir, 0755)
	// mindepth=1 excludes '.', '+' collects all file names before running the command.
	// Exclude serverless_function_source_code and .google* dir e.g. .googlebuild, .googleconfig
	command := fmt.Sprintf("find . -mindepth 1 -not -name %[1]s -prune -not -name %[2]q -prune -exec mv -t %[1]s {} +", fnSourceDir, ".google*")
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserTimingAttribution)

	fnSource := filepath.Join(ctx.ApplicationRoot(), fnSourceDir)
	pkg, err := extractPackageNameInDir(ctx, fnSource)
	if err != nil {
		return gcp.UserErrorf("error extracting package name: %v", err)
	}
	fn := fnInfo{
		Source:  fnSource,
		Target:  fnTarget,
		Package: pkg.Name,
		Imports: pkg.Imports,
	}

	if v, ok := os.LookupEnv(env.FunctionsFrameworkVersion); ok {
		functionsFrameworkVersion = v
		ctx.Logf("Use functions framework version from environment variable [%s]", functionsFrameworkVersion)
	}

	goMod := filepath.Join(fn.Source, "go.mod")
	if !ctx.FileExists(goMod) {
		// We require a go.mod file in all versions 1.14+.
		if !golang.SupportsNoGoMod(ctx) {
			return gcp.UserErrorf("function build requires go.mod file")
		}
		if err := createMainVendored(ctx, fn); err != nil {
			return err
		}
	} else if info, err := os.Stat(goMod); err == nil && info.Mode().Perm()&0200 == 0 {
		// Preempt an obscure failure mode: if go.mod is not writable then `go list -m` can fail saying:
		//     go: updates to go.sum needed, disabled by -mod=readonly
		return gcp.UserErrorf("go.mod exists but is not writable")
	} else {
		if err := createMainGoMod(ctx, fn); err != nil {
			return err
		}
	}

	ctx.AddWebProcess([]string{golang.OutBin})
	return nil
}

func createMainGoMod(ctx *gcp.Context, fn fnInfo) error {
	l := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	l.BuildEnvironment.Override("GOPATH", l.Path)
	ctx.Setenv("GOPATH", l.Path)

	// If the function source does not include a go.sum, `go list` will fail under Go 1.16+.
	if !ctx.FileExists(fn.Source, "go.sum") {
		ctx.Logf(`go.sum not found, generating using "go mod tidy"`)
		golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "tidy"}, gcp.WithWorkDir(fn.Source))
	}

	fnMod := golang.ExecWithGoproxyFallback(ctx, []string{"go", "list", "-m"}, gcp.WithWorkDir(fn.Source)).Stdout
	// golang.org/ref/mod requires that package names in a replace contains at least one dot.
	if parts := strings.Split(fnMod, "/"); len(parts) > 0 && !strings.Contains(parts[0], ".") {
		return gcp.UserErrorf("the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: %s", fnMod)
	}
	// Add the module name to the the package name, such that go build will be able to find it,
	// if a directory with the package name is not at the app root. Otherwise, assume the package is at the module root.
	if ctx.FileExists(ctx.ApplicationRoot(), fn.Package) {
		fn.Package = fmt.Sprintf("%s/%s", fnMod, fn.Package)
	} else {
		fn.Package = fnMod
	}

	ctx.Exec([]string{"go", "mod", "init", appName})
	ctx.Exec([]string{"go", "mod", "edit", "-require", fmt.Sprintf("%s@v0.0.0", fnMod)})
	ctx.Exec([]string{"go", "mod", "edit", "-replace", fmt.Sprintf("%s@v0.0.0=%s", fnMod, fn.Source)})

	// If the framework is not present in the function's go.mod, we require the current version.
	version, err := frameworkSpecifiedVersion(ctx, fn.Source)
	if err != nil {
		return fmt.Errorf("checking for functions framework dependency in go.mod: %w", err)
	}
	if version == "" {
		golang.ExecWithGoproxyFallback(ctx, []string{"go", "get", fmt.Sprintf("%s@%s", functionsFrameworkModule, functionsFrameworkVersion)}, gcp.WithUserAttribution)
		version = functionsFrameworkVersion
	}

	pluginDir := filepath.Join(fn.Source, customPluginDir)
	if ctx.FileExists(pluginDir) {
		if err := getPlugins(ctx, &fn, pluginDir); err != nil {
			return err
		}
	}

	if err := createMainGoFile(ctx, fn, filepath.Join(ctx.ApplicationRoot(), "main.go"), version); err != nil {
		return err
	}

	// Generate a go.sum entry which is required starting with Go 1.16.
	// We generate a go.mod file dynamically since the function may request a specific version of
	// the framework, in which case we want to import that version. For that reason we cannot
	// include a pre-generated go.sum file.
	golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "tidy"})
	return nil
}

// createMainVendored creates the main.go file for vendored functions.
// This should only be run for Go 1.11 and 1.13.
// Go 1.11 and 1.13 on GCF allow for vendored go.mod deployments without a go.mod file.
// Note that despite the lack of a go.mod file, this does *not* mean that these are GOPATH deployments.
// These deployments were created by running `go mod vendor` and then .gcloudignoring the go.mod file,
// so that Go versions that don't natively handle gomod vendoring would be able to pick up the vendored deps.
// n.b. later versions of Go (1.14+) handle vendored go.mod files natively, and so we just use the go.mod route there.
func createMainVendored(ctx *gcp.Context, fn fnInfo) error {
	l := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	gopath := ctx.ApplicationRoot()
	gopathSrc := filepath.Join(gopath, "src")
	ctx.MkdirAll(gopathSrc, 0755)
	l.BuildEnvironment.Override(env.Buildable, appName+"/main")
	l.BuildEnvironment.Override("GOPATH", gopath)
	ctx.Setenv("GOPATH", gopath)

	appPath := filepath.Join(gopathSrc, appName, "main")
	ctx.MkdirAll(appPath, 0755)

	// We move the function source (including any vendored deps) into GOPATH.
	ctx.Rename(fn.Source, filepath.Join(gopathSrc, fn.Package))

	fnVendoredPath := filepath.Join(gopathSrc, fn.Package, "vendor")
	fnFrameworkVendoredPath := filepath.Join(fnVendoredPath, functionsFrameworkPackage)

	// Use v0.0.0 as the requested version for go.mod-less vendored builds, since we don't know and
	// can't really tell. This won't matter for Go 1.14+, since for those we'll have a go.mod file
	// regardless.
	requestedFrameworkVersion := "v0.0.0"
	if ctx.FileExists(fnFrameworkVendoredPath) {
		ctx.Logf("Found function with vendored dependencies including functions-framework")
		ctx.Exec([]string{"cp", "-r", fnVendoredPath, appPath}, gcp.WithUserTimingAttribution)
	} else {
		// If the framework isn't in the user-provided vendor directory, we need to fetch it ourselves.
		ctx.Logf("Found function with vendored dependencies excluding functions-framework")
		ctx.Warnf("Your vendored dependencies do not contain the functions framework (%s). If there are conflicts between the vendored packages and the dependencies of the framework, you may see encounter unexpected issues.", functionsFrameworkPackage)

		// Install the functions framework. Use `go mod vendor` to do this because that allows the
		// versions of all of the framework's dependencies to be pinned as specified in the framework's
		// go.mod. Using `go get` -- the usual way to install packages in GOPATH -- downloads each
		// repository at HEAD, which can lead to breakages.
		ffDepsDir := ctx.TempDir("", "ffdeps")
		defer ctx.RemoveAll(ffDepsDir)
		cvt := filepath.Join(ctx.BuildpackRoot(), "converter", "without-framework")
		cmd := []string{
			fmt.Sprintf("cp --archive %s/. %s", cvt, ffDepsDir),
			// The only dependency is the functions framework.
			fmt.Sprintf("go mod edit -require %s@%s", functionsFrameworkModule, functionsFrameworkVersion),
			// Download the FF and its dependencies at the versions specified in the FF's go.mod.
			"go mod vendor",
			// Copy the contents of the vendor dir into GOPATH/src.
			fmt.Sprintf("cp --archive vendor/. %s", gopathSrc),
		}
		golang.ExecWithGoproxyFallback(ctx, []string{"/bin/bash", "-c", strings.Join(cmd, " && ")}, gcp.WithWorkDir(ffDepsDir), gcp.WithUserAttribution)

		// Since the user didn't pin it, we want the current version of the framework.
		requestedFrameworkVersion = functionsFrameworkVersion
	}

	return createMainGoFile(ctx, fn, filepath.Join(appPath, "main.go"), requestedFrameworkVersion)
}

func createMainGoFile(ctx *gcp.Context, fn fnInfo, main, version string) error {
	f := ctx.CreateFile(main)
	defer f.Close()

	tmpl := tmplDeclarative
	if _, ok := fn.Imports[functionsFrameworkFunctionsPackage]; !ok {
		// By default, use the v0 template.
		tmpl = tmplV0
	}

	if err := tmpl.Execute(f, fn); err != nil {
		return fmt.Errorf("executing template: %v", err)
	}
	return nil
}

// If a framework is specified, return the version. If unspecified, return an empty string.
func frameworkSpecifiedVersion(ctx *gcp.Context, fnSource string) (string, error) {
	res, err := ctx.ExecWithErr([]string{"go", "list", "-m", "-f", "{{.Version}}", functionsFrameworkModule}, gcp.WithWorkDir(fnSource))
	if err == nil {
		v := strings.TrimSpace(res.Stdout)
		ctx.Logf("Found framework version %s", v)
		return v, nil
	}
	if res != nil {
		if strings.Contains(res.Stderr, "not a known dependency") {
			ctx.Logf("functions-framework not specified in go.mod, using default")
		} else if strings.Contains(res.Stderr, "can't resolve module using the vendor directory") {
			ctx.Logf("functions-framework not found in vendor directory, using default")
		}
		return "", nil
	}
	return "", err
}

// extractPackageNameInDir builds the script that does the extraction, and then runs it with the
// specified source directory.
// The parser is dependent on the language version being used, and it's highly likely that the buildpack binary
// will be built with a different version of the language than the function deployment. Building this script ensures
// that the version of Go used to build the function app will be the same as the version used to parse it.
func extractPackageNameInDir(ctx *gcp.Context, source string) (*parsedPackage, error) {
	script := filepath.Join(ctx.BuildpackRoot(), "converter", "get_package", "main.go")
	cacheDir := ctx.TempDir("", appName)
	defer ctx.RemoveAll(cacheDir)
	stdout := ctx.Exec([]string{"go", "run", script, "-dir", source}, gcp.WithEnv("GOCACHE="+cacheDir), gcp.WithUserAttribution).Stdout

	var pkg parsedPackage
	fmt.Printf("function package: %v", stdout)
	if err := json.Unmarshal([]byte(stdout), &pkg); err != nil {
		return nil, fmt.Errorf("unable to parse function package: %v", err)
	}

	return &pkg, nil
}

// getPlugins() iterates through the "plugins" directory to get information about valid plugins
// and loads them into the fnInfo structure, which is used to render the main.go template.
// Example:
//
// |-- main.go
// |-- go.mod
// `-- plugins
//     |-- plugin-a
//     |   `-- plugin-a.go
//     `-- plugin-b
//         `-- plugin-b.go
//
// In this example, the `plugins` information would look like this:
//
//	plugins = []Plugin{
// 		{
// 			Path: "<fnInfo.Package>/plugins/plugin-a",
//			AliasName: "pluginA",
//			GetNameFunc: "pluginA.Name",
//			NewFunc: "pluginA.New()",
//		},
//		{
//			Path: "<fnInfo.Package>/plugins/plugin-b",
//			AliasName: "pluginB",
//			GetNameFunc: "pluginB.Name",
//			NewFunc: "pluginB.New()",
//		},
//	}
func getPlugins(ctx *gcp.Context, fn *fnInfo, pluginDir string) error {
	var plugins []Plugin

	pluginDirectories, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		return err
	}

	for _, plg := range pluginDirectories {
		if plg.IsDir() && strings.HasPrefix(plg.Name(), "plugin-") {
			plgAbsPath := filepath.Join(pluginDir, plg.Name())
			plgFiles, err := ioutil.ReadDir(plgAbsPath)
			if err != nil {
				ctx.Logf("failed to read directory %s: %s", plgAbsPath, err.Error())
				continue
			}
			if len(plgFiles) < 1 {
				ctx.Logf("no valid plugin file found in the %s directory", plgAbsPath)
				continue
			}

			var aliasName string
			plgNameSlice := strings.Split(plg.Name(), "-")
			aliasName = plgNameSlice[0]
			for _, s := range plgNameSlice[1:] {
				aliasName += strings.Title(s)
			}

			plugins = append(plugins, Plugin{
				Path:        filepath.Join(fn.Package, customPluginDir, plg.Name()),
				AliasName:   aliasName,
				GetNameFunc: fmt.Sprintf("%s.Name", aliasName),
				NewFunc:     fmt.Sprintf("%s.New()", aliasName),
			})
		}
	}
	fn.Plugins = plugins
	return nil
}
