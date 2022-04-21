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

package main

import (
	"reflect"
	"testing"
)

func TestParseIPRangesJSON(t *testing.T) {
	// parse a snapshot of a valid subsest of data
	const testData = `{
  "syncToken": "1649878400",
  "createDate": "2022-04-13-19-33-20",
  "prefixes": [
    {
      "ip_prefix": "3.5.140.0/22",
      "region": "ap-northeast-2",
      "service": "AMAZON",
      "network_border_group": "ap-northeast-2"
    }
  ],
  "ipv6_prefixes": [
    {
      "ipv6_prefix": "2a05:d07a:a000::/40",
      "region": "eu-south-1",
      "service": "AMAZON",
      "network_border_group": "eu-south-1"
    }
  ]
}`
	expectedParsed := &IPRangesJSON{
		Prefixes: []Prefix{
			{
				IPPrefix: "3.5.140.0/22",
				Region:   "ap-northeast-2",
				Service:  "AMAZON",
			},
		},
		IPv6Prefixes: []IPv6Prefix{
			{
				IPv6Prefix: "2a05:d07a:a000::/40",
				Region:     "eu-south-1",
				Service:    "AMAZON",
			},
		},
	}
	parsed, err := parseIPRangesJSON([]byte(testData))
	if err != nil {
		t.Fatalf("unexpected error parsing testdata: %v", err)
	}
	if !reflect.DeepEqual(expectedParsed, parsed) {
		t.Error("parsed did not match expected:")
		t.Errorf("%#v", expectedParsed)
		t.Error("parsed: ")
		t.Errorf("%#v", parsed)
		t.Fail()
	}

	// parse some bogus data
	_, err = parseIPRangesJSON([]byte(`{"prefixes": false}`))
	if err == nil {
		t.Fatal("expected error parsing garbage data but got none")
	}
}
