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

# script to run linters
set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"
source hack/tools/setup-go.sh

# build golangci-lint
cd "${REPO_ROOT}/hack/tools"
go build -o "${REPO_ROOT}"/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
cd "${REPO_ROOT}"

# lint the main module
go mod download # fetch deps first to avoid including it in timeout
"${REPO_ROOT}"/bin/golangci-lint --config "${REPO_ROOT}/hack/tools/.golangci.yml" run ./...
