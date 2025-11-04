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

func TestAZParseIPRangesJSON(t *testing.T) {
	// parse a snapshot of a valid subsest of data
	const testData = `{
  "changeNumber": 329,
  "cloud": "Public",
  "values": [
    {
      "name": "ActionGroup",
      "id": "ActionGroup",
      "properties": {
        "changeNumber": 46,
        "region": "australiacentral",
        "regionId": 0,
        "platform": "Azure",
        "systemService": "ActionGroup",
        "addressPrefixes": [
          "4.145.74.52/30",
          "4.149.254.68/30",
          "2603:1050:403::10c/126",
          "2603:1050:403:400::1f8/125"
        ],
        "networkFeatures": [
          "API",
          "NSG",
          "UDR",
          "FW"
        ]
      }
    },
    {
      "name": "ApplicationInsightsAvailability",
      "id": "ApplicationInsightsAvailability",
      "properties": {
        "changeNumber": 3,
        "region": "",
        "regionId": 0,
        "platform": "Azure",
        "systemService": "ApplicationInsightsAvailability",
        "addressPrefixes": [
          "13.86.97.224/28",
          "20.37.156.64/27"
        ],
        "networkFeatures": [
          "API",
          "NSG",
          "UDR",
          "FW"
        ]
      }
    }
  ]
}`
	expectedParsed := &AZIPRangesJSON{
		Values: []Properties{
			{
				Prefixes: AZPrefix{
					IPPrefixes: []string{
						"4.145.74.52/30",
						"4.149.254.68/30",
						"2603:1050:403::10c/126",
						"2603:1050:403:400::1f8/125",
					},
					Region: "australiacentral",
				},
			},
			{
				Prefixes: AZPrefix{
					IPPrefixes: []string{
						"13.86.97.224/28",
						"20.37.156.64/27",
					},
				},
			},
		},
	}
	parsed, err := parseAZIPRangesJSON([]byte(testData))
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
}

func TestAZRegionsToPrefixesFromData(t *testing.T) {
	t.Run("bad IPv4 prefixes", func(t *testing.T) {
		t.Parallel()
		badV4Prefixes := &AZIPRangesJSON{
			Values: []Properties{
				{
					Prefixes: AZPrefix{
						IPPrefixes: []string{
							"asdf;asdf,",
						},
					},
				},
			},
		}
		_, err := AZRegionsToPrefixesFromData(badV4Prefixes)
		if err == nil {
			t.Fatal("expected error parsing bogus prefix but got none")
		}
	})
	t.Run("bad IPv6 prefixes", func(t *testing.T) {
		t.Parallel()
		badV6Prefixes := &AZIPRangesJSON{
			Values: []Properties{
				{
					Prefixes: AZPrefix{
						IPPrefixes: []string{
							":2603:1050:403:400::1f8/125",
						},
					},
				},
			},
		}
		_, err := AZRegionsToPrefixesFromData(badV6Prefixes)
		if err == nil {
			t.Fatal("expected error parsing bogus prefix but got none")
		}
	})
}

func TestParseAZ(t *testing.T) {
	t.Run("unparsable data", func(t *testing.T) {
		t.Parallel()
		badJSON := `{"values":false}`
		_, err := parseAZ(badJSON)
		if err == nil {
			t.Fatal("expected error parsing bogus raw JSON but got none")
		}
	})
}
