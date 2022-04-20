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

// ranges2go generates a go source file with pre-parsed AWS IP ranges data.
// See also genrawdata.sh for downloading the raw data to this binary.
package main

import "os"

func main() {
	parsed, err := parseIPRangesJSON([]byte(ipRangesRaw))
	if err != nil {
		panic(err)
	}
	rtp, err := regionsToPrefixesFromData(parsed)
	if err != nil {
		panic(err)
	}
	// emit file
	f, err := os.Create("./zz_generated_range_data.go")
	if err != nil {
		panic(err)
	}
	if err := generateRangesGo(f, rtp); err != nil {
		panic(err)
	}
}
