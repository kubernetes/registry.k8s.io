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
