#!/bin/bash
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

set -o errexit -o nounset -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# normally the e2e tests would run against the staging endpoint
# this runs them against a local instance so we can test the e2e tests themselves
# and merge them even if staging is currently broken
set -x;
make archeio
bin/archeio &>"${ARTIFACTS:-./bin}"/archeio-log.txt &
trap 'kill $(jobs -p)' EXIT
make e2e-test "REGISTRY_ENDPOINT=localhost:8080"
