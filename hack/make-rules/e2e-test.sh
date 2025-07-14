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

# script to run unit / integration tests, with coverage enabled and junit xml output
set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"
source hack/tools/setup-go.sh

# build gotestsum
cd 'hack/tools'
go build -o "${REPO_ROOT}/bin/gotestsum" gotest.tools/gotestsum
cd "${REPO_ROOT}"

# run e2e tests with junit output
# TODO: because we expect relatively few packages to have e2e we only
# test those packages to limit CI noise, but this approach would work with ./...
# at the cost of reporting lots of no-test packages
# (versus the combined integration and unit testing results)
# this is also slightly faster
(
  set -x;
  "${REPO_ROOT}/bin/gotestsum" --junitfile="${REPO_ROOT}/bin/e2e-junit.xml" \
    -- '-run' '^TestE2E' '-count=1' './cmd/archeio/internal/e2e'
)

# if we are in CI, copy to the artifact upload location
if [[ -n "${ARTIFACTS:-}" ]]; then
  cp "bin/e2e-junit.xml" "${ARTIFACTS:?}/junit.xml"
fi
