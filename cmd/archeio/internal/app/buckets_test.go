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

package app

import (
	"strings"
	"testing"
)

func TestRegionToAWSRegionToHostURL(t *testing.T) {
	// Test the regions we actually support
	supportedRegions := []string{"us-east-1", "us-east-2", "us-west-1", "ap-southeast-1", "eu-central-1"}
	for _, region := range supportedRegions {
		url := awsRegionToHostURL(region, "")
		if url == "" {
			t.Fatalf("received empty string for known region %q", region)
		}
		if !strings.Contains(url, region) {
			t.Fatalf("expected URL to contain region %q, got %q", region, url)
		}
	}
	// test default region fallback
	defaultURL := "____default____"
	if url := awsRegionToHostURL("nonsensical-region", defaultURL); url != defaultURL {
		t.Fatalf("expected default URL %q for unsupported region, got %q", defaultURL, url)
	}
}

func TestBlobCache(t *testing.T) {
	bc := &blobCache{}
	bc.Put("foo", true)
	exists, found := bc.Get("foo")
	if !found || !exists {
		t.Fatal("Cache did not contain key we just put")
	}
	_, found = bc.Get("bar")
	if found {
		t.Fatal("Cache contained key we did not put")
	}
}
