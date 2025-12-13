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

# script to verify generated files
set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"
source hack/tools/setup-go.sh

tmpdir=$(mktemp -d)
trap 'rm -rf ${tmpdir?}' EXIT

# generate and compare
OUT_FILE="${tmpdir}"/zz_generated_range_data.go
export OUT_FILE
# keep excluded list in sync with hack/make-rules/codegen.sh
EXCLUDED_AWS_REGIONS="me-west-1,sa-west-1" \
DATA_DIR="${REPO_ROOT}"/pkg/net/cloudcidrs/internal/ranges2go/data \
    go run ./pkg/net/cloudcidrs/internal/ranges2go

if ! diff "${OUT_FILE}" ./pkg/net/cloudcidrs/zz_generated_range_data.go; then
    >&2 echo ""
    >&2 echo "generated file is out of date, please run 'make codegen' to regenerate"
    exit 1
fi
