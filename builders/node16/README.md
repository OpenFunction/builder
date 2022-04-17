## Build builder of nodejs version 16

### Build node16 stack

```shell
bazel run //builders/node16/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks-node16-run:v1
openfunctiondev/buildpacks-node16-build:v1
```

### Build node16 builder

```shell
bazel build //builders/node16:builder.image
```

This command creates one image:

```shell
of/node16
```

Tag and push:

```shell
docker tag of/node16 openfunction/builder-node:v2-16.13
docker push openfunction/builder-node:v2-16.13
```

### Test

```shell
bazel test //builders/node16/acceptance/...
```

Output example:

```shell
INFO: Analyzed 2 targets (1 packages loaded, 5 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 95.782s, Critical Path: 95.61s
INFO: 10 processes: 3 internal, 6 linux-sandbox, 1 local.
INFO: Build completed successfully, 10 total actions
//builders/node16/acceptance:nodejs_fn_test                              PASSED in 94.4s

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
pack build openfunctiondev/function-node:latest --builder openfunction/builder-node:v2-16.13 --env FUNC_NAME="helloWorld" --env FUNC_TYPE="http"

docker run --rm -d --name nodefunc -p8080:8080 openfunctiondev/function-node:latest
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

Definition of a ```Function``` for ```node 16``` is shown below:

```yaml
apiVersion: core.openfunction.io/v1alpha1
kind: Function
metadata:
  name: node-sample
spec:
  version: "v1.0.0"
  image: "<your registry name>/sample-node16-func:latest"
  # port: 8080 # default to 8080
  build:
    builder: "openfunctiondev/node16-builder:v1"
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
