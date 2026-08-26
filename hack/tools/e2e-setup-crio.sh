#!/usr/bin/env bash
# Copyright 2026 The Kubernetes Authors.
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

# script to ensure cri-o binaries for e2e testing
set -o errexit -o nounset -o pipefail

# cd to repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# script inputs, install dir should be versioned
readonly CRIO_VERSION="${CRIO_VERSION:?}"
readonly CRIO_INSTALL_DIR="${CRIO_INSTALL_DIR:?}"
readonly CRIO_ARCH="${CRIO_ARCH:?}"

crio_path="${CRIO_INSTALL_DIR}/bin/crio"
if [[ -f "${crio_path}" ]] && "${crio_path}" --version | grep -q "${CRIO_VERSION}"; then
    echo "Already have ${crio_path} ${CRIO_VERSION}"
else
    # download cri-o static bundle to install dir
    # bundle includes crio, crictl, conmon, crun and friends under bin/
    mkdir -p "${CRIO_INSTALL_DIR}"
    curl -sSL \
        "https://storage.googleapis.com/cri-o/artifacts/cri-o.${CRIO_ARCH}.v${CRIO_VERSION}.tar.gz" \
    | tar -C "${CRIO_INSTALL_DIR}/" -zxf - --strip-components=1
fi

# config is generated at test runtime to allow per-instance isolated paths
