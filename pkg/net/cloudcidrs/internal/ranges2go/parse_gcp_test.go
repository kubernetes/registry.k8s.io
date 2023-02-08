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

import (
	"reflect"
	"testing"
)

func TestGCPParseIPRangesJSON(t *testing.T) {
	// parse a snapshot of a valid subsest of data
	const testData = `{
  "syncToken": "1675807451971",
  "creationTime": "2023-02-07T14:04:11.9716",
  "prefixes": [{
    "ipv4Prefix": "34.80.0.0/15",
    "service": "Google Cloud",
    "scope": "asia-east1"
  }, {
    "ipv6Prefix": "2600:1900:4180::/44",
    "service": "Google Cloud",
    "scope": "us-west4"
  }]
}
`
	expectedParsed := &GCPCloudJSON{
		Prefixes: []GCPPrefix{
			{
				IPv4Prefix: "34.80.0.0/15",
				Scope:      "asia-east1",
			},
			{
				IPv6Prefix: "2600:1900:4180::/44",
				Scope:      "us-west4",
			},
		},
	}
	parsed, err := parseGCPCloudJSON([]byte(testData))
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
	_, err = parseAWSIPRangesJSON([]byte(`{"prefixes": false}`))
	if err == nil {
		t.Fatal("expected error parsing garbage data but got none")
	}
}

func TestGCPRegionsToPrefixesFromData(t *testing.T) {
	t.Run("bad IPv4 prefixes", func(t *testing.T) {
		t.Parallel()
		badV4Prefixes := &GCPCloudJSON{
			Prefixes: []GCPPrefix{
				{
					IPv4Prefix: "asdf;asdf,",
					Scope:      "us-east-1",
				},
			},
		}
		_, err := gcpRegionsToPrefixesFromData(badV4Prefixes)
		if err == nil {
			t.Fatal("expected error parsing bogus prefix but got none")
		}
	})
	t.Run("bad IPv6 prefixes", func(t *testing.T) {
		t.Parallel()
		badV6Prefixes := &GCPCloudJSON{
			Prefixes: []GCPPrefix{
				{
					IPv4Prefix: "127.0.0.1/32",
					Scope:      "us-east-1",
				},
				{
					IPv6Prefix: "asdfasdf----....",
					Scope:      "us-east-1",
				},
			},
		}
		_, err := gcpRegionsToPrefixesFromData(badV6Prefixes)
		if err == nil {
			t.Fatal("expected error parsing bogus prefix but got none")
		}
	})
}

func TestParseGCP(t *testing.T) {
	t.Run("unparsable data", func(t *testing.T) {
		t.Parallel()
		badJSON := `{"prefixes":false}`
		_, err := parseGCP(badJSON)
		if err == nil {
			t.Fatal("expected error parsing bogus raw JSON but got none")
		}
	})
}
