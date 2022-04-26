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

	"sigs.k8s.io/oci-proxy/pkg/net/cidrs/aws"
)

func TestRegionToAWSRegionToS3URL(t *testing.T) {
	// ensure all known regions return a configured bucket
	regions := aws.Regions()
	for region := range regions {
		bucket := awsRegionToS3URL(region)
		if bucket == "" {
			t.Fatalf("received empty string for known region %q bucket", region)
		}
	}
}
