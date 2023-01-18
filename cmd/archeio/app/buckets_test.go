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

	"k8s.io/registry.k8s.io/pkg/net/cidrs/aws"
)

func TestRegionToAWSRegionToS3URL(t *testing.T) {
	// ensure all known regions return a configured bucket
	regions := aws.Regions()
	for region := range regions {
		_, found := awsRegionToS3URL(region)
		if !found {
			t.Fatalf("could not find s3 mirror for known region %q", region)
		}
	}
	// ensure bogus region returns false
	if url, found := awsRegionToS3URL("nonsensical-region"); found {
		t.Fatalf("received non-empty mirror for made up region \"nonsensical-region\": %q", url)
	}
}
