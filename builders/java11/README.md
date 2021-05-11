## Build builder of java version 11

### Build java11 stack

```shell
bazel run //builders/java11/stack:build
```

This command creates two images:

```shell
openfunctiondev/buildpacks-java11-run:v1
openfunctiondev/buildpacks-java11-build:v1
```

### Build java11 builder

```shell
bazel build //builders/java11:builder.image
```

This command creates one image:

```shell
of/java11
```

Tag and push:

```shell
docker tag of/java11 <your container registry>/java11:v1
docker push <your container registry>/java11:v1
```

### Test

```shell
bazel test //builders/java11/acceptance/...
```

Output example:

```shell
INFO: Analyzed 2 targets (0 packages loaded, 0 targets configured).
INFO: Found 1 target and 1 test target...
INFO: Elapsed time: 531.375s, Critical Path: 531.11s
INFO: 7 processes: 1 internal, 5 linux-sandbox, 1 local.
INFO: Build completed successfully, 7 total actions
//builders/java11/acceptance:java_fn_test                                PASSED in 529.5s

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
cd buildpack-samples/sample-functions-framework-java-mvn/
pack build function-java --builder of/java11 --env FUNC_NAME="com.google.HelloWorld"
docker run --rm -p8080:8080 function-java
```

Visit the function:

```shell
curl http://localhost:8080
```

Output example:

```shell
Hello World!
```