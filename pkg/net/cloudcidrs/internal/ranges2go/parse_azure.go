/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"net/netip"
	"sort"
	"strings"
)

// parseAzure parses raw Azure Service Tags JSON data
// and processes it to a regionsToPrefixes map
func parseAzure(raw string) (regionsToPrefixes, error) {
	parsed, err := parseAzureServiceTagsJSON([]byte(raw))
	if err != nil {
		return nil, err
	}
	return azureRegionsToPrefixesFromData(parsed)
}

/*
	For more on this data see:
	https://www.microsoft.com/en-us/download/details.aspx?id=56519
*/

type AzureServiceTagsJSON struct {
	Values []AzureServiceTag `json:"values"`
	// changeNumber and cloud omitted
}

type AzureServiceTag struct {
	Name       string                    `json:"name"`
	Properties AzureServiceTagProperties `json:"properties"`
	// id omitted
}

type AzureServiceTagProperties struct {
	Region          string   `json:"region"`
	AddressPrefixes []string `json:"addressPrefixes"`
	// changeNumber, regionId, platform, systemService, networkFeatures omitted
}

// parseAzureServiceTagsJSON parses Azure Service Tags IP ranges JSON data
func parseAzureServiceTagsJSON(rawJSON []byte) (*AzureServiceTagsJSON, error) {
	r := &AzureServiceTagsJSON{}
	if err := json.Unmarshal(rawJSON, r); err != nil {
		return nil, err
	}
	return r, nil
}

// azureRegionsToPrefixesFromData processes the raw unmarshalled JSON into regionsToPrefixes map
func azureRegionsToPrefixesFromData(data *AzureServiceTagsJSON) (regionsToPrefixes, error) {
	// convert from Azure published structure to a map by region, parse Prefixes
	rtp := regionsToPrefixes{}
	for _, tag := range data.Values {
		// we only want the region scoped "AzureCloud.<region>" tags,
		// all other service tags are service specific subsets of these
		// the global "AzureCloud" tag has no region and duplicates the rest
		if !strings.HasPrefix(tag.Name, "AzureCloud.") || tag.Properties.Region == "" {
			continue
		}
		region := tag.Properties.Region
		// addressPrefixes contains both IPv4 and IPv6 prefixes
		for _, prefix := range tag.Properties.AddressPrefixes {
			ipPrefix, err := netip.ParsePrefix(prefix)
			if err != nil {
				return nil, err
			}
			rtp[region] = append(rtp[region], ipPrefix)
		}
	}

	// flatten
	numPrefixes := 0
	for region := range rtp {
		// this approach allows us to produce consistent generated results
		// since the ip ranges will be ordered
		sort.Slice(rtp[region], func(i, j int) bool {
			return rtp[region][i].String() < rtp[region][j].String()
		})
		rtp[region] = dedupeSortedPrefixes(rtp[region])
		numPrefixes += len(rtp[region])
	}

	return rtp, nil
}
