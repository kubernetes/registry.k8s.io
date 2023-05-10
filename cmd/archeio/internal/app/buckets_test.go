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

	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

func TestRegionToAWSRegionToHostURL(t *testing.T) {
	// ensure known regions return a configured bucket
	regions := []string{}
	for _, ipInfo := range cloudcidrs.AllIPInfos() {
		// AWS regions, excluding "GLOBAL" meta region and govcloud
		if ipInfo.Cloud == cloudcidrs.AWS &&
			ipInfo.Region != "GLOBAL" && !strings.HasPrefix(ipInfo.Region, "us-gov-") {
			regions = append(regions, ipInfo.Region)
		}
	}
	for _, region := range regions {
		url := awsRegionToHostURL(region, "")
		if url == "" {
			t.Fatalf("received empty string for known region %q", region)
		}
	}
	// test default region
	if url := awsRegionToHostURL("nonsensical-region", "____default____"); url != "____default____" {
		t.Fatalf("received non-empty URL string for made up region \"nonsensical-region\": %q", url)
	}
}

func TestBlobCache(t *testing.T) {
	bc := &blobCache{}
	bc.Put("foo")
	if !bc.Get("foo") {
		t.Fatal("Cache did not contain key we just put")
	}
	if bc.Get("bar") {
		t.Fatal("Cache contained key we did not put")
	}
}
