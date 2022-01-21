# Build builder of go version 1.16

## Version Matrix

| Builder Name | Tag description |     Functions-framework-go version     |
|---------|:---------------:|:---------------:|
| openfunction/builder-go:v0.2.2-1.16 | Buildpacks: v0.2.2, Go: 1.16 | [v0.0.0-20210628081257-4137e46a99a6](https://github.com/OpenFunction/functions-framework-go/commit/4137e46a99a6e97f1ff808b4d92ca5f76412f0cc) |
| openfunction/builder-go:v0.3.0-1.16 | Buildpacks: v0.3.0, Go: 1.16 | [v0.0.0-20210922063920-81a7b2951b8a](https://github.com/OpenFunction/functions-framework-go/commit/81a7b2951b8af0897978dcc483c1217ac98f02fb) |
| openfunction/builder-go:v0.4.0-1.16 | Buildpacks: v0.4.0, Go: 1.16 | [v0.1.1](https://github.com/OpenFunction/functions-framework-go/releases/tag/v0.1.1) |

## Build go116 stack

```shell
bazel run //builders/go116/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks-run-go:v2
openfunctiondev/buildpacks-go116-build:v2
```

## Build go116 builder

```shell
bazel build //builders/go116:builder.image
```

This command creates one image:

```shell
openfunction/builder-go
```

Tag and push:

```shell
docker tag openfunction/builder-go:v2 openfunction/builder-go:v2-1.16
docker push openfunction/builder-go:v2
docker push openfunction/builder-go:v2-1.16
```

## Test

```shell
bazel test //builders/go116/acceptance/...
```

<details>
<summary>Output example</summary>

```shell
INFO: Analyzed 2 targets (8 packages loaded, 205 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 50.633s, Critical Path: 49.87s
INFO: 10 processes: 3 internal, 6 linux-sandbox, 1 local.
INFO: Build completed successfully, 10 total actions
//builders/go116/acceptance:go_fn_test                                   PASSED in 48.1s

Executed 1 out of 1 test: 1 test passes.
INFO: Build completed successfully, 10 total actions
```

</details>

## Run locally

<details>
<summary>OpenFunction Samples</summary>

---

Download samples:

```shell
git clone https://github.com/OpenFunction/samples.git
```

Build the function:

> Add `--network host` to pack and docker command if they cannot reach internet.

```shell
cd samples/functions/Knative/hello-world-go
pack build func-helloworld-go --builder openfunction/builder-go:v2 --env FUNC_NAME="HelloWorld"  --env FUNC_CLEAR_SOURCE=true
docker run -d --env="FUNC_CONTEXT={\"name\":\"HelloWorld\",\"version\":\"v1.0.0\",\"port\":\"8080\",\"runtime\":\"Knative\"}" --rm --name func-helloworld-go -p8080:8080 func-helloworld-go
```

Visit the function:

```shell
curl http://localhost:8080
```

Output example:

```shell
hello, world!
```

</details>

<details>

<summary>GoogleCloudPlatform Samples</summary>

---

Download samples:

```shell
git clone https://github.com/GoogleCloudPlatform/buildpack-samples.git
```

Build the function:

> Add `--network host` to pack and docker command if they cannot reach internet.

```shell
cd buildpack-samples/sample-functions-framework-go/
pack build function-go --builder openfunction/builder-go:v2 --env FUNC_NAME="HelloWorld"
docker run --rm -p8080:8080 function-go
```

Visit the function:

```shell
curl http://localhost:8080
```

Output example:

```shell
hello, world
```

</details>

## Run on OpenFunction

1. [Install OpenFunction](https://github.com/OpenFunction/OpenFunction#quickstart)
2. [Run a function](https://github.com/OpenFunction/OpenFunction#sample-run-a-function)

Definition of a ```Function``` for ```go 1.16``` is shown below:

```yaml
apiVersion: core.openfunction.io/v1alpha1
kind: Function
metadata:
  name: go-sample
spec:
  version: "v1.0.0"
  image: "<your registry name>/sample-go116-func:latest"
  # port: 8080 # default to 8080
  build:
    builder: "openfunctiondev/go116-builder:v1"
    params:
      FUNC_NAME: "HelloWorld"
      FUNC_TYPE: "http"
      # FUNC_SRC: "main.py" # for python function
    srcRepo:
      url: "https://github.com/GoogleCloudPlatform/buildpack-samples.git"
      sourceSubPath: "sample-functions-framework-go"
    registry:
      url: "https://index.docker.io/v1/"
      account:
        name: "basic-user-pass"
        key: "username"
    # serving:
    # runtime: "Knative" # default to Knative
```
