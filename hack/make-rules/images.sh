#!/usr/bin/env bash

# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# script to build container images with go
set -o errexit -o nounset -o pipefail

export KO_DEFAULTBASEIMAGE='gcr.io/distroless/static-debian12@sha256:3f2b64ef97bd285e36132c684e6b2ae8f2723293d09aae046196cca64251acac'

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"
source hack/tools/setup-go.sh

# overridable list of binaries to build images for
IMAGES="${IMAGES:-cmd/archeio}"
IFS=" " read -r -a images <<< "$IMAGES"
# overridable registry to use
KO_DOCKER_REPO="${KO_DOCKER_REPO:-us-central1-docker.pkg.dev/k8s-staging-images/infra-tools}"
export KO_DOCKER_REPO
# push or local tar?
PUSH="${PUSH:-false}"
# overridable auto-tag
TAG="${TAG:-"$(date +v%Y%m%d)-$(git describe --always --dirty)"}"

# build ko
cd 'hack/tools'
go build -o "${REPO_ROOT}/bin/ko" github.com/google/ko
cd "${REPO_ROOT}"

# build images
# TODO: bake commit info into binaries consistently
for image in "${images[@]}"; do
    name="$(basename "${image}")"
    # push or local tarball
    publish_args=(--tarball=bin/"${name}".tar --push=false)
    if [[ "${PUSH}" != 'false' ]]; then
        publish_args=(--push=true)
    fi
    # specify tag
    publish_args+=(--base-import-paths --tags="${TAG}")
    # actually build
    (set -x; bin/ko publish "${publish_args[@]}" ./"${image}")
done
