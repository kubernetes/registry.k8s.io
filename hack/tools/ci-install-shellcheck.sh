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

# script to install shellcheck in CI
set -o errexit -o nounset -o pipefail

# cd to repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# get version from shellcheck script
scversion="v$(sed -nr 's/SHELLCHECK_VERSION="(.*)"/\1/p' hack/make-rules/shellcheck.sh)"
echo "Installing shellcheck ${scversion} from upstream to ensure CI version ..."
echo ""

# install xz so we can untar the upstream release
export DEBIAN_FRONTEND=noninteractive
apt-get -qq update
DEBCONF_NOWARNINGS="yes" apt-get -qq install --no-install-recommends xz-utils >/dev/null

# download and untar shellcheck into /usr/bin
wget -qO- "https://github.com/koalaman/shellcheck/releases/download/${scversion?}/shellcheck-${scversion?}.linux.x86_64.tar.xz" \
  | tar -C /usr/bin --strip-components=1 -xJ -f - "shellcheck-${scversion}/shellcheck"

# debug installed version
shellcheck --version

echo ""
echo "Done installing shellcheck ..."
echo ""
