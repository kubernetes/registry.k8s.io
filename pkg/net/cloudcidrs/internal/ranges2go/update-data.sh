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

# cd to self
cd "$(dirname "${BASH_SOURCE[0]}")"

# fetch data for each supported cloud
curl -Lo 'data/aws-ip-ranges.json' 'https://ip-ranges.amazonaws.com/ip-ranges.json'
curl -Lo 'data/gcp-cloud.json' 'https://www.gstatic.com/ipranges/cloud.json'

# Azure IP ranges are published via a download page that contains the actual download URL
# We need to extract the download URL from the confirmation page
AZURE_DOWNLOAD_PAGE='https://www.microsoft.com/en-us/download/confirmation.aspx?id=56519'
AZURE_JSON_URL=$(curl -sL "${AZURE_DOWNLOAD_PAGE}" | grep -oE 'https://[^"]*download[^"]*ServiceTags[^"]*\.json' | head -1)
if [ -z "${AZURE_JSON_URL}" ]; then
    echo "Error: Failed to extract Azure Service Tags download URL" >&2
    exit 1
fi
curl -Lo 'data/azure-cloud.json' "${AZURE_JSON_URL}"
