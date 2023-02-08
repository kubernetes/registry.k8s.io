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
	"testing"
)

func TestRegionToAWSRegionToHostURL(t *testing.T) {
	// ensure known regions return a configured bucket
	regions := []string{
		"GLOBAL", "af-south-1", "ap-east-1",
		"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
		"ap-south-1", "ap-south-2", "ap-southeast-1",
		"ap-southeast-2", "ap-southeast-3", "ap-southeast-4",
		"ap-southeast-6", "ca-central-1", "ca-west-1", "cn-north-1",
		"cn-northwest-1", "eu-central-1", "eu-central-2", "eu-north-1",
		"eu-south-1", "eu-south-2", "eu-west-1", "eu-west-2", "eu-west-3",
		"il-central-1", "me-central-1", "me-south-1", "sa-east-1", "us-east-1",
		"us-east-2", "us-gov-east-1", "us-gov-west-1", "us-west-1", "us-west-2",
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
