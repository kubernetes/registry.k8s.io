/*
Copyright 2022 The Kubernetes Authors.

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
)

// parseAZ parses raw AZ IP ranges JSON data
// and processes it to a regionsToPrefixes map
func parseAZ(raw string) (regionsToPrefixes, error) {
	parsed, err := parseAZIPRangesJSON([]byte(raw))
	if err != nil {
		return nil, err
	}
	return AZRegionsToPrefixesFromData(parsed)
}

type AZIPRangesJSON struct {
	Values []Properties `json:"values"`
}

type Properties struct {
	Prefixes AZPrefix `json:"properties"`
}

type AZPrefix struct {
	IPPrefixes []string `json:"addressPrefixes"`
	Region     string   `json:"region"`
}

// parseAZIPRangesJSON parses Azure Service Tags IP ranges JSON data
// https://learn.microsoft.com/en-us/azure/virtual-network/service-tags-overview
func parseAZIPRangesJSON(rawJSON []byte) (*AZIPRangesJSON, error) {
	r := &AZIPRangesJSON{}
	if err := json.Unmarshal(rawJSON, r); err != nil {
		return nil, err
	}
	return r, nil
}

// AZRegionsToPrefixesFromData processes the raw unmarshalled JSON into regionsToPrefixes map
func AZRegionsToPrefixesFromData(data *AZIPRangesJSON) (regionsToPrefixes, error) {
	// convert from Azure published structure to a map by region, parse Prefixes
	rtp := regionsToPrefixes{}
	for _, value := range data.Values {
		region := value.Prefixes.Region
		for _, prefix := range value.Prefixes.IPPrefixes {
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
