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

// TrieMap contains an efficient trie structure of netip.Prefix that can
// match a netip.Addr to the associated Prefix if any and return the value
// associated with it of type V.
//
// Use NewTrieMap to instantiate
//
// NOTE: This is insert-only (no delete) and insertion is *not* thread-safe.
//
// Currently this is a simple TrieMap, in the future it may have compression.
//
// See: https://vincent.bernat.ch/en/blog/2017-ipv4-route-lookup-linux
//
// For benchmarks with real data see ./aws/mapper_test.go
type TrieMap[V comparable] struct {
	// This is the real triemap, but it only maps netip.Prefix / netip.Addr : int
	// see: https://planetscale.com/blog/generics-can-make-your-go-code-slower
	// The maps below map from int in this trie to generic value type V
	//
	// This is also cheaper in many cases because int will be smaller than V
	// so we can store V only once in the map here, and int indexes into those
	// maps in the trie structure, given than many trie nodes will map to the same
	// V, as our target use-case is CIDR-to-cloud-region
	trieMap trieMap

	// simple inline bimap of int keys to V values
	//
	// the inner trie stores an int key index into keyToValue
	//
	// valueToKey is to cheapen checking if we've already inserted a given V
	// and use the same key
	keyToValue map[int]V
	valueToKey map[V]int
}

// NewTrieMap[V] returns a new, properly allocated TrieMap[V]
func NewTrieMap[V comparable]() *TrieMap[V] {
	return &TrieMap[V]{
		keyToValue: make(map[int]V),
		valueToKey: make(map[V]int),
	}
}

// Insert inserts value into TrieMap by index cidr
// You can later match a netip.Addr to value with GetIP
func (t *TrieMap[V]) Insert(cidr netip.Prefix, value V) {
	key, alreadyHave := t.valueToKey[value]
	if !alreadyHave {
		// next key = length of map
		// this structure is insert-only
		key = len(t.keyToValue)
		t.valueToKey[value] = key
		t.keyToValue[key] = value
	}
	t.trieMap.Insert(cidr, key)
}

// GetIP returns the associated value for the matching cidr if any with contains=true,
// or else the default value of V and contains=false
func (t *TrieMap[V]) GetIP(ip netip.Addr) (value V, contains bool) {
	// NOTE: this is written so as not to shadow contains locally
	// and so we can use value as a default-value for V without
	// another variable, using the name also to document the return
	key, c := t.trieMap.GetIP(ip)
	contains = c
	if !contains {
		return
	}
	value = t.keyToValue[key]
	return
}

// trieMap is the core implementation, but it only stores netip.Prefix : int
type trieMap struct {
	// surely ipv4 and ipv6 will be enough in our lifetime?
	ipv4Root *trieNode
	ipv6Root *trieNode
}

// TODO: path compression
type trieNode struct {
	// children for 0 and 1 bits
	child0 *trieNode
	child1 *trieNode
	// both of these values will be set together or not set
	// so we place them in a sub struct to save memory at the cost
	// of chasing one additional pointer per trie node checked
	value *nodeValue
}

type nodeValue struct {
	cidr netip.Prefix
	key  int
}

func (t *trieMap) Insert(cidr netip.Prefix, key int) {
	if cidr.Addr().Is4() {
		t.insertIPV4(cidr, key)
	} else {
		t.insertIPV6(cidr, key)
	}
}

func (t *trieMap) insertIPV4(cidr netip.Prefix, key int) {
	// ensure root node
	if t.ipv4Root == nil {
		t.ipv4Root = &trieNode{}
	}

	// walk bits high to low, inserting matching ip path up to mask bits
	curr := t.ipv4Root
	ip := cidr.Addr().As4()
	// first cast to uint32 for fast bit access
	// NOTE: IP addresses are big endian, so the low bits are in the last byte
	ipInt := uint32(ip[3]) | uint32(ip[2])<<8 | uint32(ip[1])<<16 | uint32(ip[0])<<24
	bits := cidr.Bits()
	for i := 31; i >= (32 - bits); i-- {
		if (ipInt & (uint32(1) << i)) != 0 {
			if curr.child1 == nil {
				curr.child1 = &trieNode{}
			}
			curr = curr.child1
		} else {
			if curr.child0 == nil {
				curr.child0 = &trieNode{}
			}
			curr = curr.child0
		}
	}
	curr.value = &nodeValue{
		cidr: cidr,
		key:  key,
	}
}

func (t *trieMap) insertIPV6(cidr netip.Prefix, key int) {
	// ensure root node
	if t.ipv6Root == nil {
		t.ipv6Root = &trieNode{}
	}

	// walk bits high to low, inserting matching ip path up to mask bits
	curr := t.ipv6Root
	ip := cidr.Addr().As16()
	bits := cidr.Bits()
	// first cast ip to two uint64 for fast bit access
	// NOTE: IP addresses are big endian, so the low bits are in the last byte
	ipLo := uint64(ip[15]) | uint64(ip[14])<<8 | uint64(ip[13])<<16 | uint64(ip[12])<<24 |
		uint64(ip[11])<<32 | uint64(ip[10])<<40 | uint64(ip[9])<<48 | uint64(ip[8])<<56
	ipHi := uint64(ip[7]) | uint64(ip[6])<<8 | uint64(ip[5])<<16 | uint64(ip[4])<<24 |
		uint64(ip[3])<<32 | uint64(ip[2])<<40 | uint64(ip[1])<<48 | uint64(ip[0])<<56
	for i := 127; i >= (128 - bits); i-- {
		bit := false
		if i > 63 {
			bit = (ipHi & (uint64(1) << (i - 64))) != 0
		} else {
			bit = (ipLo & (uint64(1) << i)) != 0
		}
		if bit {
			if curr.child1 == nil {
				curr.child1 = &trieNode{}
			}
			curr = curr.child1
		} else {
			if curr.child0 == nil {
				curr.child0 = &trieNode{}
			}
			curr = curr.child0
		}
	}
	curr.value = &nodeValue{
		cidr: cidr,
		key:  key,
	}
}

func (t *trieMap) GetIP(ip netip.Addr) (int, bool) {
	if ip.Is4() {
		return t.getIPv4(ip)
	}
	return t.getIPv6(ip)
}

func (t *trieMap) getIPv4(addr netip.Addr) (int, bool) {
	// check the root first
	curr := t.ipv4Root
	if curr == nil {
		return -1, false
	}
	if curr.value != nil && curr.value.cidr.Contains(addr) {
		return curr.value.key, true
	}
	// walk IP bits high to low, checking if current node matches
	ip := addr.As4()
	// first cast to uint32 for fast bit access
	// NOTE: IP addresses are big endian, so the low bits are in the last byte
	ipInt := uint32(ip[3]) | uint32(ip[2])<<8 | uint32(ip[1])<<16 | uint32(ip[0])<<24
	for i := 31; i >= 0; i-- {
		// walk based on current address bit
		if (ipInt & (uint32(1) << i)) != 0 {
			if curr.child1 != nil {
				curr = curr.child1
			} else {
				// dead end
				break
			}
		} else {
			if curr.child0 != nil {
				curr = curr.child0
			} else {
				// dead end
				break
			}
		}
		// check for a match in the current node
		if curr.value != nil && curr.value.cidr.Contains(addr) {
			return curr.value.key, true
		}
	}
	return -1, false
}

func (t *trieMap) getIPv6(addr netip.Addr) (int, bool) {
	// check the root first
	curr := t.ipv6Root
	if curr == nil {
		return -1, false
	}
	if curr.value != nil && curr.value.cidr.Contains(addr) {
		return curr.value.key, true
	}
	// walk IP bits high to low, checking if current node matches
	// first cast ip to two uint64 for fast bit access
	ip := addr.As16()
	// NOTE: IP addresses are big endian, so the low bits are in the last byte
	ipLo := uint64(ip[15]) | uint64(ip[14])<<8 | uint64(ip[13])<<16 | uint64(ip[12])<<24 |
		uint64(ip[11])<<32 | uint64(ip[10])<<40 | uint64(ip[9])<<48 | uint64(ip[8])<<56
	ipHi := uint64(ip[7]) | uint64(ip[6])<<8 | uint64(ip[5])<<16 | uint64(ip[4])<<24 |
		uint64(ip[3])<<32 | uint64(ip[2])<<40 | uint64(ip[1])<<48 | uint64(ip[0])<<56
	for i := 127; i >= 0; i-- {
		bit := false
		if i > 63 {
			bit = (ipHi & (uint64(1) << (i - 64))) != 0
		} else {
			bit = (ipLo & (uint64(1) << i)) != 0
		}
		// walk based on current address bit
		if bit {
			if curr.child1 != nil {
				curr = curr.child1
			} else {
				// dead end
				break
			}
		} else {
			if curr.child0 != nil {
				curr = curr.child0
			} else {
				// dead end
				break
			}
		}
		// check for a match in the current node
		if curr.value != nil && curr.value.cidr.Contains(addr) {
			return curr.value.key, true
		}
	}
	return -1, false
}
