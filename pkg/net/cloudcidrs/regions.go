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

// Regions returns a set-like map of all known AWS regions
// based on the same underlying data as the rest of this package
func Regions() map[string]bool {
	regions := map[string]bool{}
	for region := range regionToRanges {
		regions[region] = true
	}
	return regions
}
