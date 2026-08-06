/*
Copyright 2026 The Kubernetes Authors.

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
	"net/netip"
	"reflect"
	"testing"
)

const azureTestData = `{
  "changeNumber": 356,
  "cloud": "Public",
  "values": [
    {
      "name": "AzureCloud",
      "id": "AzureCloud",
      "properties": {
        "changeNumber": 356,
        "region": "",
        "regionId": 0,
        "platform": "Azure",
        "systemService": "",
        "addressPrefixes": [
          "4.128.0.0/12"
        ],
        "networkFeatures": []
      }
    },
    {
      "name": "AzureCloud.eastus",
      "id": "AzureCloud.eastus",
      "properties": {
        "changeNumber": 42,
        "region": "eastus",
        "regionId": 32,
        "platform": "Azure",
        "systemService": "",
        "addressPrefixes": [
          "4.156.0.0/15",
          "2603:1030:210::/48",
          "4.156.0.0/15"
        ],
        "networkFeatures": []
      }
    },
    {
      "name": "AzureCloud.westeurope",
      "id": "AzureCloud.westeurope",
      "properties": {
        "changeNumber": 41,
        "region": "westeurope",
        "regionId": 18,
        "platform": "Azure",
        "systemService": "",
        "addressPrefixes": [
          "13.69.0.0/17"
        ],
        "networkFeatures": []
      }
    },
    {
      "name": "ActionGroup",
      "id": "ActionGroup",
      "properties": {
        "changeNumber": 24,
        "region": "",
        "regionId": 0,
        "platform": "Azure",
        "systemService": "ActionGroup",
        "addressPrefixes": [
          "4.145.74.52/30"
        ],
        "networkFeatures": []
      }
    }
  ]
}`

func TestParseAzure(t *testing.T) {
	rtp, err := parseAzure(azureTestData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// the global AzureCloud tag and service specific tags must be skipped,
	// duplicate prefixes must be deduped, and results must be sorted
	expected := regionsToPrefixes{
		"eastus": []netip.Prefix{
			netip.MustParsePrefix("2603:1030:210::/48"),
			netip.MustParsePrefix("4.156.0.0/15"),
		},
		"westeurope": []netip.Prefix{
			netip.MustParsePrefix("13.69.0.0/17"),
		},
	}
	if !reflect.DeepEqual(rtp, expected) {
		t.Fatalf("result does not match, got: %v expected: %v", rtp, expected)
	}
}

func TestParseAzureBadJSON(t *testing.T) {
	if _, err := parseAzure(`{"values": [}`); err == nil {
		t.Fatal("expected error parsing invalid JSON but got none")
	}
}

func TestParseAzureBadPrefix(t *testing.T) {
	const badPrefixData = `{
  "values": [
    {
      "name": "AzureCloud.eastus",
      "properties": {
        "region": "eastus",
        "addressPrefixes": [
          "not-a-prefix/99"
        ]
      }
    }
  ]
}`
	if _, err := parseAzure(badPrefixData); err == nil {
		t.Fatal("expected error parsing invalid prefix but got none")
	}
}
