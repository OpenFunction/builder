#!/bin/bash

with_stack=false
java_version=11
docker_registry=openfunction
create_builder_only=false
build_image=""
run_image=""
out_image=""
push_image=false

build_java() {
  VERSION_ARRAY=(11 16 17 18)

  # shellcheck disable=SC2199
  # shellcheck disable=SC2053
  if [[ ${VERSION_ARRAY[@]/${java_version}/} == ${VERSION_ARRAY[@]} ]]; then
    echo "unsupported java version $java_version"
    exit
  fi

  dir=builders/java/java$java_version
  mkdir "$dir"
  ls builders/java/ | grep -v 'java.*' | xargs -i cp -r builders/java/{} "$dir"/

  sed -i -e "s/< JAVA_VERSION >/${java_version}/g" "$dir"/builder.toml
  sed -i -e "s/< JAVA_VERSION >/${java_version}/g" "$dir"/BUILD.bazel
  sed -i -e "s/< JAVA_VERSION >/${java_version}/g" "$dir"/stack/build.sh
  sed -i -e "s/< JAVA_VERSION >/${java_version}/g" "$dir"/stack/parent.Dockerfile

  sed -i -e "s/< REGISTRY >/${docker_registry}/g" "$dir"/builder.toml
  sed -i -e "s/< REGISTRY >/${docker_registry}/g" "$dir"/BUILD.bazel
  sed -i -e "s/< REGISTRY >/${docker_registry}/g" "$dir"/stack/build.sh

  if [[ -n "$run_image" ]]; then
    sed -ri "s/(run-image = )[^\n]*/\1\"${run_image}\"/" "$dir"/builder.toml
  else
    run_image="$docker_registry"/buildpacks-java"$java_version"-run:v1
  fi

  if [[ -n "$build_image" ]]; then
    sed -ri "s/(build-image = )[^\n]*/\1\"${build_image}\"/" "$dir"/builder.toml
  else
    build_image="$docker_registry"/buildpacks-java"$java_version"-build:v1
  fi

  if [[ -n "$out_image" ]]; then
    sed -ri "s/(image = )[^\n]*/\1\"${out_image}\"/" "$dir"/BUILD.bazel
  else
    out_image="$docker_registry"/builder-java:v2-"$java_version"
  fi

  # only create a builder directory, not build image
  if [[ "$create_builder_only" == "true" ]]; then
    exit
  fi

  if [[ "$with_stack" == "true" ]]; then
    bazel run //builders/java/java"${java_version}"/stack:build
  fi

  bazel build //builders/java/java"${java_version}":builder.image --action_env https_proxy="${https_proxy}"

  if [[ "$push_image" == "true" ]]; then
    if [[ "$with_stack" == "true" ]]; then
      docker push $run_image
      docker push $build_image
    fi
    docker push $out_image
  fi

  rm -rf "$dir"
}

TEMP=$(getopt -o an:p --long all,with-stack,java-version:,docker-registry:,create-builder-only,build-image:,run-image:,out-image:,push-image \
  -- "$@")

# Note the quotes around `$TEMP`: they are essential!
eval set -- "$TEMP"
while true; do
  case "$1" in
  --with-stack)
    with_stack=true
    shift
    ;;
  --java-version)
    java_version=$2
    shift
    ;;
  --docker-registry)
    docker_registry=$2
    shift
    ;;
  --create-builder-only)
    create_builder_only=true
    shift
    ;;
  --run-image)
    run_image=$2
    shift
    ;;
  --build-image)
    build_image=$2
    shift
    ;;
  --out-image)
    out_image=$2
    shift
    ;;
  --push-image)
    push_image=true
    shift
    ;;
  --)
    shift
    break
    ;;
  *)
    shift
    ;;
  esac
done

language="$1"
if [[ "$language" == "java" ]]; then
  build_java
else
  echo "unsupported language $language"
fi
