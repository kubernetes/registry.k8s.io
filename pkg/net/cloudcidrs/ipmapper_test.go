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

package cloudcidrs

import (
	"net/netip"
	"testing"

	"k8s.io/registry.k8s.io/pkg/net/cidrs"
)

type testCase struct {
	Addr           netip.Addr
	ExpectedRegion string
}

// some known IPs and their regions
var testCasesIPv4 = []testCase{
	{Addr: netip.MustParseAddr("35.180.1.1"), ExpectedRegion: "eu-west-3"},
	{Addr: netip.MustParseAddr("35.250.1.1"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("35.0.1.1"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("52.94.76.1"), ExpectedRegion: "us-west-2"},
	{Addr: netip.MustParseAddr("52.94.77.1"), ExpectedRegion: "us-west-2"},
	{Addr: netip.MustParseAddr("52.93.127.172"), ExpectedRegion: "us-east-1"},
}
var testCasesIPv6 = []testCase{
	// ipv6
	{Addr: netip.MustParseAddr("2400:6500:0:9::2"), ExpectedRegion: "ap-southeast-3"},
	{Addr: netip.MustParseAddr("2400:6500:0:9::1"), ExpectedRegion: "ap-southeast-3"},
	{Addr: netip.MustParseAddr("2400:6500:0:9::3"), ExpectedRegion: "ap-southeast-3"},
	{Addr: netip.MustParseAddr("2600:1f01:4874::47"), ExpectedRegion: "us-west-2"},
	{Addr: netip.MustParseAddr("2400:6500:0:9::100"), ExpectedRegion: ""},
}

// NOTE: we need to append to an empty slice so we do not modify the existing slices
// append may re-use the first existing slice ...
var allTestCases = append(append([]testCase{}, testCasesIPv4...), testCasesIPv6...)

func TestNewIPMapper(t *testing.T) {
	mapper := NewIPMapper()
	for i := range allTestCases {
		tc := allTestCases[i]
		t.Run(tc.Addr.String(), func(t *testing.T) {
			r, matched := mapper.GetIP(tc.Addr)
			expectMatched := tc.ExpectedRegion != ""
			if matched != expectMatched || r.Region != tc.ExpectedRegion {
				t.Fatalf(
					"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
					tc.Addr, r.Region, matched, tc.ExpectedRegion, expectMatched,
				)
			}
		})
	}
}

/*  for benchmarking memory / init time */

func BenchmarkNewIPMapper(b *testing.B) {
	for n := 0; n < b.N; n++ {
		mapper := NewIPMapper()
		// get any address just to prevent mapper being optimized out
		mapper.GetIP(allTestCases[0].Addr)
	}
}

func BenchmarkNewegionBruteForce(b *testing.B) {
	for n := 0; n < b.N; n++ {
		mapper := cidrs.NewBruteForceMapper(regionToRanges)
		// get any address just to prevent mapper being optimized out
		mapper.GetIP(allTestCases[0].Addr)
	}
}

/* for benchmarking matching time */

func BenchmarkRegionTrieMapIPv4(b *testing.B) {
	mapper := NewIPMapper()
	for n := 0; n < b.N; n++ {
		tc := testCasesIPv4[n%len(testCasesIPv4)]
		r, matched := mapper.GetIP(tc.Addr)
		expectMatched := tc.ExpectedRegion != ""
		if matched != expectMatched || r.Region != tc.ExpectedRegion {
			b.Fatalf(
				"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
				tc.Addr, r.Region, matched, tc.ExpectedRegion, expectMatched,
			)
		}
	}
}

func BenchmarkRegionTrieMapIPv6(b *testing.B) {
	mapper := NewIPMapper()
	for n := 0; n < b.N; n++ {
		tc := testCasesIPv6[n%len(testCasesIPv6)]
		r, matched := mapper.GetIP(tc.Addr)
		expectMatched := tc.ExpectedRegion != ""
		if matched != expectMatched || r.Region != tc.ExpectedRegion {
			b.Fatalf(
				"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
				tc.Addr, r.Region, matched, tc.ExpectedRegion, expectMatched,
			)
		}
	}
}

func BenchmarkRegionBruteForceIPv4(b *testing.B) {
	mapper := cidrs.NewBruteForceMapper(regionToRanges)
	for n := 0; n < b.N; n++ {
		tc := testCasesIPv4[n%len(testCasesIPv4)]
		r, matched := mapper.GetIP(tc.Addr)
		expectMatched := tc.ExpectedRegion != ""
		if matched != expectMatched || r.Region != tc.ExpectedRegion {
			b.Fatalf(
				"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
				tc.Addr, r.Region, matched, tc.ExpectedRegion, expectMatched,
			)
		}
	}
}

func BenchmarkRegionBruteForceIPv6(b *testing.B) {
	mapper := cidrs.NewBruteForceMapper(regionToRanges)
	for n := 0; n < b.N; n++ {
		tc := testCasesIPv6[n%len(testCasesIPv6)]
		r, matched := mapper.GetIP(tc.Addr)
		expectMatched := tc.ExpectedRegion != ""
		if matched != expectMatched || r.Region != tc.ExpectedRegion {
			b.Fatalf(
				"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
				tc.Addr, r.Region, matched, tc.ExpectedRegion, expectMatched,
			)
		}
	}
}
