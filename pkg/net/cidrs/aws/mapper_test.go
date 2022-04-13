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

package aws

import (
	"net/netip"
	"testing"
)

func TestNewAWSRegionMapper(t *testing.T) {
	testCases := []struct {
		Addr           netip.Addr
		ExpectedRegion string
	}{
		// some known IPs and their regions
		{Addr: netip.MustParseAddr("35.180.1.1"), ExpectedRegion: "eu-west-3"},
		{Addr: netip.MustParseAddr("35.250.1.1"), ExpectedRegion: ""},
		{Addr: netip.MustParseAddr("35.0.1.1"), ExpectedRegion: ""},
		{Addr: netip.MustParseAddr("52.94.76.1"), ExpectedRegion: "us-west-2"},
		{Addr: netip.MustParseAddr("52.94.77.1"), ExpectedRegion: "us-west-2"},
		{Addr: netip.MustParseAddr("52.93.127.172"), ExpectedRegion: "us-east-1"},
		// ipv6
		{Addr: netip.MustParseAddr("2400:6500:0:9::2"), ExpectedRegion: "ap-southeast-3"},
		{Addr: netip.MustParseAddr("2400:6500:0:9::1"), ExpectedRegion: "ap-southeast-3"},
		{Addr: netip.MustParseAddr("2400:6500:0:9::3"), ExpectedRegion: "ap-southeast-3"},
		{Addr: netip.MustParseAddr("2600:1f01:4874::47"), ExpectedRegion: "us-west-2"},
		{Addr: netip.MustParseAddr("2400:6500:0:9::100"), ExpectedRegion: ""},
	}
	mapper := NewAWSRegionMapper()
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Addr.String(), func(t *testing.T) {
			region, matched := mapper.GetIP(tc.Addr)
			expectMatched := tc.ExpectedRegion != ""
			if matched != expectMatched || region != tc.ExpectedRegion {
				t.Fatalf(
					"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
					tc.Addr, region, matched, tc.ExpectedRegion, expectMatched,
				)
			}
		})
	}
}
