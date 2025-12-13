#!/usr/bin/env bash

# Copyright 2025 The Kubernetes Authors.
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

# script to run all codegen
set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

source hack/tools/setup-go.sh

echo "Downloading AWS & GCP IP ranges data..."
curl -fLo 'pkg/net/cloudcidrs/internal/ranges2go/data/aws-ip-ranges.json' 'https://ip-ranges.amazonaws.com/ip-ranges.json'
curl -fLo 'pkg/net/cloudcidrs/internal/ranges2go/data/gcp-cloud.json' 'https://www.gstatic.com/ipranges/cloud.json'

# AWS adds IP ranges for unreleased regions which we want to exclude
EXCLUDED_AWS_REGIONS="me-west-1,sa-west-1" \
OUT_FILE=pkg/net/cloudcidrs/zz_generated_range_data.go \
DATA_DIR=pkg/net/cloudcidrs/internal/ranges2go/data \
    go run ./pkg/net/cloudcidrs/internal/ranges2go
