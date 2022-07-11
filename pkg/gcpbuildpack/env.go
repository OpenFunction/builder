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

package gcpbuildpack

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpacks/libcnb"
)

// SetFunctionsEnvVars sets launch-time functions environment variables.
func (ctx *Context) SetFunctionsEnvVars(l *libcnb.Layer) {
	if target := os.Getenv(env.FunctionTarget); target != "" {
		l.LaunchEnvironment.Default(env.FunctionTarget, target)
		l.LaunchEnvironment.Default(env.FunctionTargetLaunch, target)
	} else {
		ctx.Exit(1, UserErrorf("required env var %s not found", env.FunctionTarget))
	}

	if signature, ok := os.LookupEnv(env.FunctionSignatureType); ok {
		l.LaunchEnvironment.Default(env.FunctionSignatureTypeLaunch, signature)
	}

	if source, ok := os.LookupEnv(env.FunctionSource); ok {
		l.LaunchEnvironment.Default(env.FunctionSourceLaunch, source)
	}
}
