#!/usr/bin/env bash
# Copyright 2023 The Kubernetes Authors.
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

# script to ensure containerd binaries for e2e testing
set -o errexit -o nounset -o pipefail

# cd to repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# script inputs, install dir should be versioned
readonly CONTAINERD_VERSION="${CONTAINERD_VERSION:?}"
readonly CONTAINERD_INSTALL_DIR="${CONTAINERD_INSTALL_DIR:?}"
readonly CONTAINERD_ARCH="${CONTAINERD_ARCH:?}"

containerd_path="${CONTAINERD_INSTALL_DIR}/containerd"
if [[ -f "${containerd_path}" ]] && "${containerd_path}" --version | grep -q "${CONTAINERD_VERSION}"; then
    echo "Already have ${containerd_path} ${CONTAINERD_VERSION}"
else
    # downlod containerd to bindir
    mkdir -p "${CONTAINERD_INSTALL_DIR}"
    curl -sSL \
        "https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-${CONTAINERD_ARCH}.tar.gz" \
    | tar -C "${CONTAINERD_INSTALL_DIR}/" -zxvf - --strip-components=1
fi

# generate config for current user
cat <<EOF >"${CONTAINERD_INSTALL_DIR}"/containerd-config.toml
# own socket as as current user.
# we will be running as this user and only fetching
[grpc]
  uid = $(id -u)
  gid = $(id -g)
EOF
