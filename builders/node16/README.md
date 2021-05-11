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
docker tag of/node16 <your container registry>/node16:v1
docker push <your container registry>/node16:v1
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
pack build function-node --builder of/node16 --env FUNC_NAME="helloWorld"
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