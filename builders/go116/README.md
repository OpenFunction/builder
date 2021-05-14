## Build builder of go version 1.15

### Build go116 stack

```shell
bazel run //builders/go116/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks-go116-run:v1
openfunctiondev/buildpacks-go116-build:v1
```

### Build go116 builder

```shell
bazel build //builders/go116:builder.image
```

This command creates one image:

```shell
of/go116
```

Tag and push:

```shell
docker tag of/go116 <your container registry>/go116:v1
docker push <your container registry>/go116:v1
```

### Test

```shell
bazel test //builders/go116/acceptance/...
```

Output example:

```shell
INFO: Analyzed 2 targets (0 packages loaded, 0 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 36.640s, Critical Path: 36.47s
INFO: 7 processes: 1 internal, 5 linux-sandbox, 1 local.
INFO: Build completed successfully, 7 total actions
//builders/go116/acceptance:go_fn_test                                   PASSED in 35.4s

Executed 1 out of 1 test: 1 test passes.
INFO: Build completed successfully, 7 total actions
```

### Run

Download gcp samples:

```shell
git clone https://github.com/GoogleCloudPlatform/buildpack-samples.git
```

Build the function:

> Add `--network host` to pack and docker command if they cannot reach internet.

```shell
cd buildpack-samples/sample-functions-framework-go/
pack build function-go --builder of/go116 --env FUNC_NAME="HelloWorld"
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

### Run on OpenFunction

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