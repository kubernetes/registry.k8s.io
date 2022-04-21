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

TAG="${TAG:-"$(date +v%Y%m%d)-$(git describe --always --dirty)"}"
SERVICE_BASENAME="${SERVICE_BASENAME:-k8s-infra-oci-proxy}"
IMAGE_REPO="${IMAGE_REPO:-gcr.io/k8s-staging-infra-tools/archeio}"
PROJECT="${PROJECT:-k8s-infra-oci-proxy}"

REGIONS=(
    us-central1
    us-west1
)

for REGION in "${REGIONS[@]}"; do
    gcloud --project="${PROJECT}" \
        run services update "${SERVICE_BASENAME}-${REGION}" \
        --image "${IMAGE_REPO}:${TAG}" \
        --region "${REGION}"
done
