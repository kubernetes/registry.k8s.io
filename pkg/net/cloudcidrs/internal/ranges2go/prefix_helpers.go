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

import "net/netip"

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
