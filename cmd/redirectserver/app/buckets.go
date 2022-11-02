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

// awsRegionToS3URL returns the base S3 bucket URL for an OCI layer blob given the AWS region
//
// blobs in the buckets should be stored at /containers/images/sha256:$hash
func awsRegionToS3URL(region string) string {
	switch region {
	// each of these has the region in which we have a bucket listed first
	// and then additional regions we're mapping to that bucket
	// based roughly on physical adjacency (and therefore _presumed_ latency)
	//
	// if you add a bucket, add a case for the region it is in, and consider
	// shifting other regions that do not have their own bucket

	// US East (N. Virginia)
	case "us-east-1", "sa-east-1", "us-gov-east-1", "GLOBAL":
		return "https://prod-artifacts-k8s-io-us-east-1.s3.dualstack.us-east-1.amazonaws.com"
	// US East (Ohio)
	case "us-east-2", "ca-central-1":
		return "https://prod-artifacts-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com"
	// US West (N. California)
	case "us-west-1", "us-gov-west-1":
		return "https://prod-artifacts-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com"
	// US West (Oregon)
	case "us-west-2", "ca-west-1":
		return "https://prod-artifacts-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com"
	// Asia Pacific (Mumbai)
	case "ap-south-1", "ap-south-2", "me-south-1", "me-central-1":
		return "https://prod-artifacts-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com"
	// Asia Pacific (Tokyo)
	case "ap-northeast-1", "ap-northeast-2", "ap-northeast-3":
		return "https://prod-artifacts-k8s-io-ap-northeast-1.s3.dualstack.ap-northeast-1.amazonaws.com"
	// Asia Pacific (Singapore)
	case "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4", "ap-southeast-6", "ap-east-1", "cn-northwest-1", "cn-north-1":
		return "https://prod-artifacts-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com"
	// Europe (Frankfurt)
	case "eu-central-1", "eu-central-2", "eu-south-1", "eu-south-2", "il-central-1":
		return "https://prod-artifacts-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com"
	// Europe (Ireland)
	case "eu-west-1", "af-south-1":
		return "https://prod-artifacts-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com"
	// Europe (London)
	case "eu-west-2", "eu-west-3", "eu-north-1":
		return "https://prod-artifacts-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com"
	default:
		// TestRegionToAWSRegionToS3URL checks we return a non-empty result for all regions
		// that this app knows about
		//
		// we will not attempt to route to a region we do now know about
		//
		// if we see empty string returned, then we've failed to account for all regions
		//
		// we want to precompute the mapping for all regions
		return ""
	}
}
