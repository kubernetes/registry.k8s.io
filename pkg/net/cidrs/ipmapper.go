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

import "net/netip"

// IPMapper represents an type can perform a get on a map of netip.Addr to
// value V, typically implemented against netip.Prefix data
//
// See TrieMap for an efficient implementation
type IPMapper[V comparable] interface {
	GetIP(ip netip.Addr) (value V, matches bool)
}
