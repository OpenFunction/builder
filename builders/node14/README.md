## Build builder of nodejs version 14

### Build node14 stack

```shell
bazel run //builders/node14/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks-node14-run:v1
openfunctiondev/buildpacks-node14-build:v1
```

### Build node14 builder

```shell
bazel build //builders/node14:builder.image
```

This command creates one image:

```shell
of/node14
```

Tag and push:

```shell
docker tag of/node14 <your container registry>/node14:v1
docker push <your container registry>/node14:v1
```

### Test

```shell
bazel test //builders/node14/acceptance/...
```

Output example:

```shell
INFO: Analyzed 2 targets (1 packages loaded, 5 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 97.652s, Critical Path: 97.46s
INFO: 10 processes: 3 internal, 6 linux-sandbox, 1 local.
INFO: Build completed successfully, 10 total actions
//builders/node14/acceptance:nodejs_fn_test                              PASSED in 96.3s

Executed 1 out of 1 test: 1 test passes.
INFO: Build completed successfully, 10 total actions
```

### Run

Download gcp samples:

```shell
git clone https://github.com/GoogleCloudPlatform/buildpack-samples.git
```

Build the function:

> Add `--network host` to pack and docker command if they cannot reach internet.

```shell
cd buildpack-samples/sample-functions-framework-node/
pack build function-node --builder of/node14 --env FUNC_NAME="helloWorld"
docker run --rm -p8080:8080 function-node
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

Definition of a ```Function``` for ```node 14``` is shown below:

```yaml
apiVersion: core.openfunction.io/v1alpha1
kind: Function
metadata:
  name: node-sample
spec:
  version: "v1.0.0"
  image: "<your registry name>/sample-node14-func:latest"
  # port: 8080 # default to 8080
  build:
    builder: "openfunctiondev/node14-builder:v1"
    params:
      FUNC_NAME: "helloWorld"
      FUNC_TYPE: "http"
      # FUNC_SRC: "main.py" # for python function
    srcRepo:
      url: "https://github.com/GoogleCloudPlatform/buildpack-samples.git"
      sourceSubPath: "sample-functions-framework-node"
    registry:
      url: "https://index.docker.io/v1/"
      account:
        name: "basic-user-pass"
        key: "username"
    # serving:
    # runtime: "Knative" # default to Knative
```
