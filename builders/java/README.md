## Build builder of java

Use the follow command to build a java builder.

```shell
hack/build.sh java --with-stack --java-version < java_version > --docker-registry < registry >
```

This command will create three image:

```shell
< REGISTRY >/buildpacks-java< java_version >-run:v1
< REGISTRY >/buildpacks-java< java_version >-build:v1
< REGISTRY >/builder-java:v2-< java_version >
```

Parameters

- `with-stack` - If specified, it will also build the build image and run image too.
- `java-version` - The version of java, known value are 11, 16, 17, 18.
- `docker-registry` - The docker registry where the image pushed to, default is `openfunction`.
- `create-builder-only` - Only create the builder directory, and do not build image. The builder directory will be under `builders/java`.
- `build-image` - The image used to build builder.
- `run-image` - The image used to run the function.
- `out-image` - The name of generated builder image.
- `push-image` - If specified, it will push the image to the registry.