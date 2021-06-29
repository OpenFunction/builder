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
	"errors"
	"fmt"
	userfunction "{{.Package}}"
	"github.com/OpenFunction/functions-framework-go/functionframeworks"
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"log"
	"net/http"
)

func register(fn interface{}) error {
	ctx := context.Background()
	if fnHTTP, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
		if err := functionframeworks.RegisterHTTPFunction(ctx, fnHTTP); err != nil {
			return fmt.Errorf("Function failed to register: %v\n", err)
		}
	} else if fnCloudEvent, ok := fn.(func(context.Context, cloudevents.Event) error); ok {
		if err := functionframeworks.RegisterCloudEventFunction(ctx, fnCloudEvent); err != nil {
			return fmt.Errorf("Function failed to register: %v\n", err)
		}
	} else if fnOpenFunction, ok := fn.(func(*ofctx.OpenFunctionContext, []byte) int); ok {
		if err := functionframeworks.RegisterOpenFunction(ctx, fnOpenFunction); err != nil {
			return fmt.Errorf("Function failed to register: %v\n", err)
		}
	} else {
		err := errors.New("unrecognized function")
		return fmt.Errorf("Function failed to register: %v\n", err)
	}
	return nil
}

func main() {
	if err := register(userfunction.{{.Target}}); err != nil {
		log.Fatalf("Failed to register: %v\n", err)
	}

	if err := functionframeworks.Start(); err != nil {
		log.Fatalf("Failed to start: %v\n", err)
	}
}`
