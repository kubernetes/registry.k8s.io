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

set -o errexit -o nounset -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# if we're in cloudbuild then we might want to change the project to point
# at where we're deploying instead of deploying from
if [[ -n "${CLOUDBUILD_SET_PROJECT:-}" ]]; then
    gcloud config set project "${CLOUDBUILD_SET_PROJECT:?}"
fi

# make sure we have a k8s.io clone for the prod terraform
k8sio_dir="$(cd "${REPO_ROOT}"/../k8s.io && pwd -P)"
if [[ ! -d "${k8sio_dir}" ]]; then
    >&2 echo "Deploying requires a github.com/kubernetes/k8s.io clone at ./../k8s.io"
    >&2 echo "FAIL"
    exit 1
fi

# install crane and get current image digest
TAG="${TAG:-"$(date +v%Y%m%d)-$(git describe --always --dirty)"}"
# TODO: this can't actually be overridden currently
# the terraform always uses the default here
IMAGE_REPO="${IMAGE_REPO:-us-central1-docker.pkg.dev/k8s-staging-images/infra-tools/archeio}"
GOBIN="${REPO_ROOT}/bin" go install github.com/google/go-containerregistry/cmd/crane@latest
IMAGE_DIGEST="${IMAGE_DIGEST:-$(bin/crane digest "${IMAGE_REPO}:${TAG}")}"
export IMAGE_DIGEST

# cd to staging terraform and apply
cd "${k8sio_dir}"/infra/gcp/terraform/k8s-infra-oci-proxy
# use tfswitch to control terraform version based on sources, if available
if command -v tfswitch >/dev/null; then
    tfswitch
fi
terraform -v
terraform init
# NOTE: this must use :? expansion to ensure we will not run with unset variables
(set -x; terraform apply -auto-approve -var digest="${IMAGE_DIGEST:?}")
