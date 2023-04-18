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

# CI script to run shellcheck
set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# we will be installing under bin_dir if necessary, and re-using if possible
bin_dir="${REPO_ROOT}/bin"
export PATH="${bin_dir}:${PATH}"

# required version for this script, if not installed on the host already we will
# install it under bin/
SHELLCHECK_VERSION="0.8.0"

# Find all shell scripts excluding:
# - Anything git-ignored - No need to lint untracked files.
# - ./.git/* - Ignore anything in the git object store.
# - ./hack/third_party/* - Ignore vendored scripts.
# - ./bin/* - No need to lint output directories.
all_shell_scripts=()
while IFS=$'\n' read -r script;
  do git check-ignore -q "$script" || all_shell_scripts+=("$script");
done < <(grep -irl '#!.*sh' . | grep -Ev '^(\./\.git/)|(\./hack/third_party/)|(\./bin/)')

# common arguments we'll pass to shellcheck
SHELLCHECK_OPTIONS=(
  # allow following sourced files that are not specified in the command,
  # we need this because we specify one file at a time in order to trivially
  # detect which files are failing
  '--external-sources'
  # disabled lint codes
  # 2330 - disabled due to https://github.com/koalaman/shellcheck/issues/1162
  '--exclude=2230'
  # 2126 - disabled because grep -c exits error when there are zero matches,
  # unlike grep | wc -l
  '--exclude=2126'
  # set colorized output
  '--color=auto'
)

# detect if the host machine has the required shellcheck version installed
# if so, we will use that instead.
HAVE_SHELLCHECK=false
if command -v shellcheck &>/dev/null; then
  detected_version="$(shellcheck --version | grep 'version: .*')"
  if [[ "${detected_version}" = "version: ${SHELLCHECK_VERSION}" ]]; then
    HAVE_SHELLCHECK=true
  fi
fi

# install shellcheck to bin/ if missing or the wrong version
if ! ${HAVE_SHELLCHECK}; then
  echo "Installing shellcheck v${SHELLCHECK_VERSION} under bin/ ..." >&2
  # in CI we can install xz so we can untar the upstream release
  # otherwise tell the user they must install xz or shellcheck
  if ! command -v xz &>/dev/null; then
    if [[ -n "${PROW_JOB_ID}" ]]; then
      export DEBIAN_FRONTEND=noninteractive
      apt-get -qq update
      DEBCONF_NOWARNINGS="yes" apt-get -qq install --no-install-recommends xz-utils >/dev/null
    else
      echo "xz is required to install shellcheck in bin/!" >&2
      echo "either install xz or install shellcheck v${SHELLCHECK_VERSION}" >&2
      exit 1
    fi
  fi
  os=$(uname | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  # TODO: shellcheck currently only has x86_64 binaries on macOS, but those will work on M1
  if [[ "${os}" == 'darwin' ]]; then
    arch='x86_64'
  fi
  mkdir -p "${bin_dir}"
  # download and untar shellcheck into bin_dir
  curl -sSL "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION?}/shellcheck-v${SHELLCHECK_VERSION?}.${os}.${arch}.tar.xz" \
    | tar -C "${bin_dir}" --strip-components=1 -xJ -f - "shellcheck-v${SHELLCHECK_VERSION}/shellcheck"
  # debug newly setup version
  shellcheck --version >&2
fi


# lint all scripts
res=0
shellcheck "${SHELLCHECK_OPTIONS[@]}" "${all_shell_scripts[@]}" >&2 || res=$?
# print a message based on the result
if [ $res -eq 0 ]; then
  echo 'Congratulations! All shell files are passing lint :-)'
else
  {
    echo
    echo 'Please review the above warnings. You can test via "./hack/verify/shellcheck.sh"'
    echo 'If the above warnings do not make sense, you can exempt this warning with a comment'
    echo ' (if your reviewer is okay with it).'
    echo 'In general please prefer to fix the error, we have already disabled specific lints'
    echo ' that the project chooses to ignore.'
    echo 'See: https://github.com/koalaman/shellcheck/wiki/Ignore#ignoring-one-specific-instance-in-a-file'
    echo
  } >&2
  exit 1
fi

# preserve the result
exit $res
