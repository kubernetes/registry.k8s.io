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
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// awsRegionToHostURL returns the base S3 bucket URL for an OCI layer blob given the AWS region
//
// blobs in the buckets should be stored at /containers/images/sha256:$hash
func awsRegionToHostURL(region, defaultURL string) string {
	switch region {
	// each of these has the region in which we have a bucket listed first
	// and then additional regions we're mapping to that bucket
	// based roughly on physical adjacency (and therefore _presumed_ latency)
	//
	// As of late 2025, we don't have access to cn-northwest-1 or cn-north-1 regions as they are part of the aws-cn partition.
	// So we are mapping them to ap-east-1(Hong Kong) for now.
	// aws ec2 describe-regions --all-regions --query "Regions[].RegionName" --output json | jq .[] | awk '{print $0","}' | sort --version-sort

	// Africa (Cape Town)
	case "af-south-1":
		return "https://prod-registry-k8s-io-af-south-1.s3.dualstack.af-south-1.amazonaws.com"
	// Asia Pacific (Hong Kong) and China Regions
	case "ap-east-1", "cn-northwest-1", "cn-north-1":
		return "https://prod-registry-k8s-io-ap-east-1.s3.dualstack.ap-east-1.amazonaws.com"
	// Asia Pacific (Taipei)
	case "ap-east-2":
		return "https://prod-registry-k8s-io-ap-east-1.s3.dualstack.ap-east-1.amazonaws.com"
	// Asia Pacific (Tokyo)
	case "ap-northeast-1":
		return "https://prod-registry-k8s-io-ap-northeast-1.s3.dualstack.ap-northeast-1.amazonaws.com"
	// Asia Pacific (Seoul)
	case "ap-northeast-2":
		return "https://prod-registry-k8s-io-ap-northeast-2.s3.dualstack.ap-northeast-2.amazonaws.com"
	// Asia Pacific (Osaka)
	case "ap-northeast-3":
		return "https://prod-registry-k8s-io-ap-northeast-3.s3.dualstack.ap-northeast-3.amazonaws.com"
	// Asia Pacific (Singapore)
	case "ap-southeast-1":
		return "https://prod-registry-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com"
	// Asia Pacific (Sydney)
	case "ap-southeast-2":
		return "https://prod-registry-k8s-io-ap-southeast-2.s3.dualstack.ap-southeast-2.amazonaws.com"
	// Asia Pacific (Jakarta)
	case "ap-southeast-3":
		return "https://prod-registry-k8s-io-ap-southeast-3.s3.dualstack.ap-southeast-3.amazonaws.com"
	// Asia Pacific (Melbourne)
	case "ap-southeast-4":
		return "https://prod-registry-k8s-io-ap-southeast-4.s3.dualstack.ap-southeast-4.amazonaws.com"
	// Asia Pacific (Singapore)
	case "ap-southeast-5":
		return "https://prod-registry-k8s-io-ap-southeast-5.s3.dualstack.ap-southeast-5.amazonaws.com"
	// Asia Pacific (New Zealand)
	case "ap-southeast-6":
		return "https://prod-registry-k8s-io-ap-southeast-6.s3.dualstack.ap-southeast-6.amazonaws.com"
	// Asia Pacific (Thailand)
	case "ap-southeast-7":
		return "https://prod-registry-k8s-io-ap-southeast-7.s3.dualstack.ap-southeast-7.amazonaws.com"
	// Asia Pacific (Mumbai)
	case "ap-south-1":
		return "https://prod-registry-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com"
	// Asia Pacific (Hyderabad)
	case "ap-south-2":
		return "https://prod-registry-k8s-io-ap-south-2.s3.dualstack.ap-south-2.amazonaws.com"
	// Canada (Central)
	case "ca-central-1":
		return "https://prod-registry-k8s-io-ca-central-1.s3.dualstack.ca-central-1.amazonaws.com"
	// Canada (Calgary)
	case "ca-west-1":
		return "https://prod-registry-k8s-io-ca-west-1.s3.dualstack.ca-west-1.amazonaws.com"
	// Europe (Frankfurt)
	case "eu-central-1":
		return "https://prod-registry-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com"
	// Europe (Zurich)
	case "eu-central-2":
		return "https://prod-registry-k8s-io-eu-central-2.s3.dualstack.eu-central-2.amazonaws.com"
	// Europe (Stockholm)
	case "eu-north-1":
		return "https://prod-registry-k8s-io-eu-north-1.s3.dualstack.eu-north-1.amazonaws.com"
	// Europe (Milan)
	case "eu-south-1":
		return "https://prod-registry-k8s-io-eu-south-1.s3.dualstack.eu-south-1.amazonaws.com"
	// Europe (Spain)
	case "eu-south-2":
		return "https://prod-registry-k8s-io-eu-south-2.s3.dualstack.eu-south-2.amazonaws.com"
	// Europe (Ireland)
	case "eu-west-1":
		return "https://prod-registry-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com"
	// Europe (London)
	case "eu-west-2":
		return "https://767373bbdcb8270361b96548387bf2a9ad0d48758c35-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com"
	// Europe (Paris)
	case "eu-west-3":
		return "https://prod-registry-k8s-io-eu-west-3.s3.dualstack.eu-west-3.amazonaws.com"
	// Israel (Tel Aviv)
	case "il-central-1":
		return "https://prod-registry-k8s-io-il-central-1.s3.dualstack.il-central-1.amazonaws.com"
	// Middle East (UAE)
	case "me-central-1":
		return "https://prod-registry-k8s-io-me-central-1.s3.dualstack.me-central-1.amazonaws.com"
	// Middle East (Bahrain)
	case "me-south-1":
		return "https://prod-registry-k8s-io-me-south-1.s3.dualstack.me-south-1.amazonaws.com"
	// Mexico (Central)
	case "mx-central-1":
		return "https://prod-registry-k8s-io-mx-central-1.s3.dualstack.mx-central-1.amazonaws.com"
	// South America (São Paulo)
	case "sa-east-1":
		return "https://prod-registry-k8s-io-sa-east-1.s3.dualstack.sa-east-1.amazonaws.com"
	// US East (N. Virginia)
	case "us-east-1":
		return "https://prod-registry-k8s-io-us-east-1.s3.dualstack.us-east-1.amazonaws.com"
	// US East (Ohio)
	case "us-east-2":
		return "https://prod-registry-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com"
	// US West (N. California)
	case "us-west-1":
		return "https://prod-registry-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com"
	// US West (Oregon)
	case "us-west-2":
		return "https://prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com"
	default:
		return defaultURL
	}
}

// blobChecker are used to check if a blob exists, possibly with caching
type blobChecker interface {
	// BlobExists should check that blobURL exists
	// bucket and layerHash may be used for caching purposes
	BlobExists(blobURL string) bool
}

// cachedBlobChecker just performs an HTTP HEAD check against the blob
//
// TODO: potentially replace with a caching implementation
// should be plenty fast for now, HTTP HEAD on s3 is cheap
type cachedBlobChecker struct {
	blobCache
}

func newCachedBlobChecker() *cachedBlobChecker {
	return &cachedBlobChecker{}
}

type blobCache struct {
	m sync.Map
}

func (b *blobCache) Get(blobURL string) bool {
	_, exists := b.m.Load(blobURL)
	return exists
}

func (b *blobCache) Put(blobURL string) {
	b.m.Store(blobURL, struct{}{})
}

func (c *cachedBlobChecker) BlobExists(blobURL string) bool {
	if c.blobCache.Get(blobURL) {
		klog.V(3).InfoS("blob existence cache hit", "url", blobURL)
		return true
	}
	klog.V(3).InfoS("blob existence cache miss", "url", blobURL)
	// NOTE: this client will still share http.DefaultTransport
	// We do not wish to share the rest of the client state currently
	client := &http.Client{
		// ensure sensible timeouts
		Timeout: time.Second * 5,
	}
	r, err := client.Head(blobURL)
	// fallback to assuming blob is unavailable on errors
	if err != nil {
		return false
	}
	r.Body.Close()
	// if the blob exists it HEAD should return 200 OK
	// this is true for S3 and for OCI registries
	if r.StatusCode == http.StatusOK {
		c.blobCache.Put(blobURL)
		return true
	}
	return false
}
