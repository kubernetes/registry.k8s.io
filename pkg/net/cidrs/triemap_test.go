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

package cidrs

import (
	"net/netip"
	"testing"
)

func TestTrieMap(t *testing.T) {
	trieMap := NewTrieMap[string]()
	for value, cidrs := range testCIDRS {
		for _, cidr := range cidrs {
			trieMap.Insert(cidr, value)
		}
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Addr.String(), func(t *testing.T) {
			t.Parallel()
			// NOTE: we set region == "" for no-contains
			expectedContains := tc.ExpectedRegion != ""
			ip := tc.Addr
			region, contains := trieMap.GetIP(ip)
			if contains != expectedContains || region != tc.ExpectedRegion {
				t.Fatalf(
					"result does not match for %v, got: (%q, %t) expected: (%q, %t)",
					ip, region, contains, tc.ExpectedRegion, expectedContains,
				)
			}
		})
	}
}

func TestTrieMapEmpty(t *testing.T) {
	trieMap := NewTrieMap[string]()
	v, contains := trieMap.GetIP(netip.MustParseAddr("127.0.0.1"))
	if contains || v != "" {
		t.Fatalf("empty TrieMap should not contain anything")
	}
	v, contains = trieMap.GetIP(netip.MustParseAddr("::1"))
	if contains || v != "" {
		t.Fatalf("empty TrieMap should not contain anything")
	}
}

func TestTrieMapSlashZero(t *testing.T) {
	// test the ??? case that we insert into the root with a /0
	trieMap := NewTrieMap[string]()
	trieMap.Insert(netip.MustParsePrefix("0.0.0.0/0"), "all-ipv4")
	trieMap.Insert(netip.MustParsePrefix("::/0"), "all-ipv6")
	v, contains := trieMap.GetIP(netip.MustParseAddr("127.0.0.1"))
	if !contains || v != "all-ipv4" {
		t.Fatalf("TrieMap failed to match IPv4 with all IPs in one /0")
	}
	v, contains = trieMap.GetIP(netip.MustParseAddr("::1"))
	if !contains || v != "all-ipv6" {
		t.Fatalf("TrieMap failed to match IPv6 with all IPs in one /0")
	}
}
