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

import "k8s.io/registry.k8s.io/pkg/net/cidrs"

// NewIPMapper returns cidrs.IPMapper populated with cloud region info
// for the clouds we have resources for, currently GCP and AWS
func NewIPMapper() cidrs.IPMapper[IPInfo] {
	t := cidrs.NewTrieMap[IPInfo]()
	for info, cidrs := range regionToRanges {
		for _, cidr := range cidrs {
			t.Insert(cidr, info)
		}
	}
	return t
}

// AllIPInfos returns a slice of all known results that a NewIPMapper could
// return
func AllIPInfos() []IPInfo {
	r := make([]IPInfo, 0, len(regionToRanges))
	for v := range regionToRanges {
		r = append(r, v)
	}
	return r
}
