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

/*
	For more on these datatypes see:
	https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html
*/

type IPRangesJSON struct {
	Prefixes     []Prefix     `json:"prefixes"`
	IPv6Prefixes []IPv6Prefix `json:"ipv6_prefixes"`
	// syncToken and createDate omitted
}

type Prefix struct {
	IPPrefix string `json:"ip_prefix"`
	Region   string `json:"region"`
	Service  string `json:"service"`
	// network_border_group omitted
}

type IPv6Prefix struct {
	IPv6Prefix string `json:"ipv6_prefix"`
	Region     string `json:"region"`
	Service    string `json:"service"`
	// network_border_group omitted
}

// parseIPRangesJSON parse AWS IP ranges JSON data
// https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html
func parseIPRangesJSON(rawJSON []byte) (*IPRangesJSON, error) {
	r := &IPRangesJSON{}
	if err := json.Unmarshal(rawJSON, r); err != nil {
		return nil, err
	}
	return r, nil
}

// regionsToPrefixes is the structure we process the JSON into
type regionsToPrefixes map[string][]netip.Prefix

// regionsToPrefixesFromData processes the raw unmarshalled JSON into regionsToPrefixes map
func regionsToPrefixesFromData(data *IPRangesJSON) (regionsToPrefixes, error) {
	// convert from AWS published structure to a map by region, parse Prefixes
	rtp := regionsToPrefixes{}
	for _, prefix := range data.Prefixes {
		region := prefix.Region
		ipPrefix, err := netip.ParsePrefix(prefix.IPPrefix)
		if err != nil {
			return nil, err
		}
		rtp[region] = append(rtp[region], ipPrefix)
	}
	for _, prefix := range data.IPv6Prefixes {
		region := prefix.Region
		ipPrefix, err := netip.ParsePrefix(prefix.IPv6Prefix)
		if err != nil {
			return nil, err
		}
		rtp[region] = append(rtp[region], ipPrefix)
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

func dedupeSortedPrefixes(s []netip.Prefix) []netip.Prefix {
	l := len(s)
	// nothing to do for <= 1
	if l <= 1 {
		return s
	}
	// for 1..len(s) if previous entry does not match, keep current
	j := 0
	for i := 1; i < l; i++ {
		if s[i].String() != s[i-1].String() {
			s[j] = s[i]
			j++
		}
	}
	return s[0:j]
}
