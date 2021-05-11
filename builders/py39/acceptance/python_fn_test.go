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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptancePythonFn(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "function with framework",
			App:        "with_framework",
			Path:       "/testFunction",
			Env:        []string{"FUNC_NAME=testFunction"},
			MustUse:    []string{pythonPIP, pythonFF},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailuresPythonFn(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "missing framework file",
			App:       "with_framework",
			Env:       []string{"FUNC_NAME=testFunction", "FUNC_SRC=func.py"},
			MustMatch: `FUNC_SRC specified file "func.py" but it does not exist`,
		},
		{
			Name:      "missing main.py",
			App:       "custom_file",
			Env:       []string{"FUNC_NAME=testFunction"},
			MustMatch: "missing main.py and FUNC_SRC not specified. Either create the function in main.py or specify FUNC_SRC to point to the file that contains the function",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
