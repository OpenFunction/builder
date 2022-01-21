#!/usr/bin/env bash
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The build.sh script builds stack images for the gcp/base builder.
#
# The script builds the following two images:
#   gcr.io/buildpacks/gcp/run:$tag
#   gcr.io/buildpacks/gcp/build:$tag
#
# It also validates that the build image includes all required licenses.
#
# Usage:
#   ./build.sh <path to self> <path to licenses.tar>

set -euo pipefail

# Convenient way to find the runfiles directory containing the Dockerfiles.
# $0 is in a different directory top-level directory.
DIR="$(dirname "$1")"
LICENSES="$2"

TAG="v2"

# Extract licenses.tar because it is symlinked, which Docker does not support.
readonly TEMP="$(mktemp -d)"
trap "rm -rf $TEMP" EXIT

echo "> Extracting licenses tar"
mkdir -p "$TEMP/licenses"
tar xf "$LICENSES" -C "$TEMP/licenses"

echo "> Building base go116common image"
docker build -t "go116common" - < "${DIR}/parent.Dockerfile"
echo "> Building base go116run image"
docker build -t "go116run" - < "${DIR}/run.busybox.Dockerfile"
echo "> Building openfunctiondev/buildpacks-run-go:$TAG"
docker build --build-arg "from_image=go116run" -t "openfunctiondev/buildpacks-run-go:$TAG" - < "${DIR}/run.Dockerfile"
echo "> Building openfunctiondev/buildpacks-go116-build:$TAG"
docker build --build-arg "from_image=go116common" -t "openfunctiondev/buildpacks-go116-build:$TAG" -f "${DIR}/build.Dockerfile" "${TEMP}"