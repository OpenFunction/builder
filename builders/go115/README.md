## Build builder of go version 1.15

### Build go115 stack

```shell
bazel run //builders/go115/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks/go115/run:v1
openfunctiondev/buildpacks/go115/build:v1
```

### Build go115 builder

```shell
bazel build //builders/go115:builder.image
```

This command creates two images:

```shell
of/go115
```

Tag and push:

```shell
docker tag of/go115 <your container registry>/go115:v1
docker push <your container registry>/go115:v1
```

### Test

```shell
bazel test //builders/go115/acceptance/...
```

Output example:

```shell
INFO: Analyzed 2 targets (0 packages loaded, 0 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 36.640s, Critical Path: 36.47s
INFO: 7 processes: 1 internal, 5 linux-sandbox, 1 local.
INFO: Build completed successfully, 7 total actions
//builders/go115/acceptance:go_fn_test                                   PASSED in 35.4s

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
pack build function-go --builder of/go115 --env FUNC_TARGET="HelloWorld"
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