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

package main

const mainTextTemplate = `// Binary main file implements an HTTP server that loads and runs user's code
// on incoming HTTP requests.
// As this file must compile statically alongside the user code, this file
// will be copied into the function image and the 'FUNCTION_TARGET' and
// 'FUNCTION_PACKAGE' strings will be replaced by the relevant function and
// package names. That edited file will then be compiled as with the user's
// function code to produce an executable app binary that launches the HTTP
// server.
package main

import (
	"context"

	"k8s.io/klog/v2"
	"github.com/OpenFunction/functions-framework-go/framework"
	"github.com/OpenFunction/functions-framework-go/plugin"
	userfunction "{{.Package}}"

	{{- range $Plugin := .Plugins }}
	{{ $Plugin.AliasName }} "{{ $Plugin.Path -}}"
	{{- end }}
)

func main() {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		klog.Exit(err)
	}
	fwk.RegisterPlugins(getLocalPlugins())
	if err := fwk.Register(ctx, userfunction.{{.Target}}); err != nil {
		klog.Exit(err)
	}
	if err := fwk.Start(ctx); err != nil {
		klog.Exit(err)
	}
}

func getLocalPlugins() map[string]plugin.Plugin {
	localPlugins := map[string]plugin.Plugin{
		{{- range $Plugin := .Plugins }}
		{{ $Plugin.GetNameFunc }}: {{ $Plugin.NewFunc }},
		{{- end }}
	}

	if len(localPlugins) == 0 {
		return nil
	} else {
		return localPlugins
	}
}`
