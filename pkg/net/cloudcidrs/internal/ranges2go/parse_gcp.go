/*
Copyright 2023 The Kubernetes Authors.

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
	"errors"
	"net/netip"
	"sort"
)

// parseGCP parses raw GCP cloud.json data
// and processes it to a regionsToPrefixes map
func parseGCP(raw string) (regionsToPrefixes, error) {
	parsed, err := parseGCPCloudJSON([]byte(raw))
	if err != nil {
		return nil, err
	}
	return gcpRegionsToPrefixesFromData(parsed)
}

type GCPCloudJSON struct {
	Prefixes []GCPPrefix `json:"prefixes"`
	// syncToken and createDate omitted
}

type GCPPrefix struct {
	IPv4Prefix string `json:"ipv4Prefix"`
	IPv6Prefix string `json:"ipv6Prefix"`
	Scope      string `json:"scope"`
	// service omitted
}

// parseGCPCloudJSON parses GCP cloud.json IP ranges JSON data
func parseGCPCloudJSON(rawJSON []byte) (*GCPCloudJSON, error) {
	r := &GCPCloudJSON{}
	if err := json.Unmarshal(rawJSON, r); err != nil {
		return nil, err
	}
	return r, nil
}

// gcpRegionsToPrefixesFromData processes the raw unmarshalled JSON into regionsToPrefixes map
func gcpRegionsToPrefixesFromData(data *GCPCloudJSON) (regionsToPrefixes, error) {
	// convert from AWS published structure to a map by region, parse Prefixes
	rtp := regionsToPrefixes{}
	for _, prefix := range data.Prefixes {
		region := prefix.Scope
		if prefix.IPv4Prefix != "" {
			ipPrefix, err := netip.ParsePrefix(prefix.IPv4Prefix)
			if err != nil {
				return nil, err
			}
			rtp[region] = append(rtp[region], ipPrefix)
		} else if prefix.IPv6Prefix != "" {
			ipPrefix, err := netip.ParsePrefix(prefix.IPv6Prefix)
			if err != nil {
				return nil, err
			}
			rtp[region] = append(rtp[region], ipPrefix)
		} else {
			return nil, errors.New("unexpected entry with no ipv4Prefix or ipv6Prefix")
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
