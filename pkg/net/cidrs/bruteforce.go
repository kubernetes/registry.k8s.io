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
)

// NewBruteForceMapper returns a naive IPMapper that loops over the map and
// compares all netip.Prefix
//
// This type exists purely for testing and benchmarking
func NewBruteForceMapper[V comparable](mapping map[netip.Prefix]V) IPMapper[V] {
	return &bruteForceMapper[V]{
		mapping: mapping,
	}
}

type bruteForceMapper[V comparable] struct {
	mapping map[netip.Prefix]V
}

func (b *bruteForceMapper[V]) GetIP(addr netip.Addr) (value V, matched bool) {
	for cidr, v := range b.mapping {
		if cidr.Contains(addr) {
			return v, true
		}
	}
	return
}
