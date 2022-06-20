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

// Binary dotnet/runtime buildpack detects .NET applications
// and install the corresponding version of .NET runtime.
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	sdkLayerName             = "sdk"
	runtimeLayerName         = "runtime"
	aspnetRuntimeURL         = "https://dotnetcli.azureedge.net/dotnet/aspnetcore/Runtime/%[1]s/aspnetcore-runtime-%[1]s-linux-x64.tar.gz"
	uncachedAspnetRuntimeURL = "https://dotnetcli.blob.core.windows.net/dotnet/aspnetcore/Runtime/%[1]s/aspnetcore-runtime-%[1]s-linux-x64.tar.gz"
	versionKey               = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("dotnet"); result != nil {
		return result, nil
	}

	if files := dotnet.ProjectFiles(ctx, "."); len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}
	rtCfgs, err := dotnet.RuntimeConfigJSONFiles(".")
	if err != nil {
		return nil, fmt.Errorf("finding runtimeconfig.json: %w", err)
	}
	if len(rtCfgs) > 0 {
		return gcp.OptIn("found at least one runtimeconfig.json"), nil
	}

	return gcp.OptOut("no project files or .dll files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	sdkVersion, err := dotnet.GetSDKVersion(ctx)
	if err != nil {
		return err
	}
	isDevMode, err := env.IsDevMode()
	if err != nil {
		return fmt.Errorf("checking if dev mode is enabled: %w", err)
	}
	if err := buildSDKLayer(ctx, sdkVersion, isDevMode); err != nil {
		return fmt.Errorf("building the sdk layer: %w", err)
	}
	if !isDevMode {
		if err := buildRuntimeLayer(ctx, sdkVersion); err != nil {
			return fmt.Errorf("building the runtime layer: %w", err)
		}
	}
	return nil
}

func buildSDKLayer(ctx *gcp.Context, version string, isDevMode bool) error {
	// Keep the SDK layer for launch in devmode because we use `dotnet watch`.
	sdkl := ctx.Layer(sdkLayerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)

	sdkMetaVersion := ctx.GetMetadata(sdkl, versionKey)
	cacheHitValue := fmt.Sprintf("version:%s,devMode:%t", version, isDevMode)
	if cacheHitValue == sdkMetaVersion {
		ctx.CacheHit(sdkLayerName)
		ctx.Logf(".NET SDK cache hit, skipping installation.")
		return nil
	}
	
	ctx.CacheMiss(sdkLayerName)
	ctx.ClearLayer(sdkl)
	dlAndInstallSDK(ctx, sdkl, version, isDevMode)
	ctx.SetMetadata(sdkl, versionKey, cacheHitValue)
	return nil
}

func dlAndInstallSDK(ctx *gcp.Context, sdkl *libcnb.Layer, version string, isDevMode bool) error {
	if _, err := runtime.InstallTarballIfNotCached(ctx, runtime.DotnetSDK, version, sdkl); err != nil {
		return err
	}
	setSDKEnvVars(sdkl, isDevMode)
	return nil
}

func setSDKEnvVars(sdkl *libcnb.Layer, isDevMode bool) {
	if isDevMode {
		setSDKEnvVarsDevMode(sdkl)
	} else {
		setSDKEnvVarsForBuild(sdkl)
	}
}

// setSDKEnvVarsDevMode sets the env vars for dev mode. In dev mode, the full
// SDK is present at launch time and the runtime layer is not created.
func setSDKEnvVarsDevMode(sdkl *libcnb.Layer) {
	sdkl.SharedEnvironment.Default("DOTNET_ROOT", sdkl.Path)
	sdkl.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), sdkl.Path)
	sdkl.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")
}

// setSDKEnvVarsForBuild sets the SDK variables needed at build time. The SDK
// layer is only present for the build and the runtime layer is present in the launch
// image.
func setSDKEnvVarsForBuild(sdkl *libcnb.Layer) {
	sdkl.BuildEnvironment.Default("DOTNET_ROOT", sdkl.Path)
	sdkl.BuildEnvironment.Prepend("PATH", string(os.PathListSeparator), sdkl.Path)
}

func buildRuntimeLayer(ctx *gcp.Context, sdkVersion string) error {
	rtVersion, err := release.GetRuntimeVersionForSDKVersion(client.New(), sdkVersion)
	if err != nil {
		return err
	}
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     runtimeLayerName,
		Metadata: map[string]interface{}{"version": rtVersion},
		Launch:   true,
		Build:    true,
	})
	rtl := ctx.Layer(runtimeLayerName, gcp.CacheLayer, gcp.LaunchLayer)
	
	rtMetaVersion := ctx.GetMetadata(rtl, versionKey)
	if rtVersion == rtMetaVersion {
		ctx.CacheHit(runtimeLayerName)
		ctx.Logf(".NET runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(runtimeLayerName)
	ctx.ClearLayer(rtl)
	if err := dlAndInstallRuntime(ctx, rtl, rtVersion); err != nil {
		return err
	}
	return nil
}

func dlAndInstallRuntime(ctx *gcp.Context, rtl *libcnb.Layer, version string) error {
	aspnetcoreRuntimeArchiveURL, err := aspnetcoreRuntimeArchiveURL(ctx, version)
	if err != nil {
		return err
	}
	ctx.Logf("Installing ASP.NET Core Runtime v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", aspnetcoreRuntimeArchiveURL, rtl.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	ctx.SetMetadata(rtl, versionKey, version)
	rtl.LaunchEnvironment.Default("DOTNET_ROOT", rtl.Path)
	rtl.LaunchEnvironment.Prepend("PATH", string(os.PathListSeparator), rtl.Path)
	rtl.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")
	return nil
}

func aspnetcoreRuntimeArchiveURL(ctx *gcp.Context, version string) (string, error) {
	url := fmt.Sprintf(aspnetRuntimeURL, version)
	code := ctx.HTTPStatus(url)

	if code == http.StatusOK {
		return url,nil
	}
	url = fmt.Sprintf(uncachedAspnetRuntimeURL, version)
	code = ctx.HTTPStatus(url)
	
	if code != http.StatusOK {
		return "", gcp.UserErrorf("runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, url, code, env.RuntimeVersion)
	}
	return url, nil
}
