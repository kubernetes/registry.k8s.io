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
	// if you add a bucket, add a case for the region it is in, and consider
	// shifting other regions that do not have their own bucket

	// US East (N. Virginia)
	case "us-east-1", "sa-east-1", "mx-central-1":
		return "https://prod-registry-k8s-io-us-east-1.s3.dualstack.us-east-1.amazonaws.com"
	// US East (Ohio)
	case "us-east-2", "ca-central-1":
		return "https://prod-registry-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com"
	// US West (N. California)
	case "us-west-1", "sa-west-1":
		return "https://prod-registry-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com"
	// US West (Oregon)
	case "us-west-2", "ca-west-1":
		return "https://prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com"
	// Asia Pacific (Mumbai)
	case "ap-south-1", "ap-south-2", "me-south-1", "me-central-1", "me-west-1":
		return "https://prod-registry-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com"
	// Asia Pacific (Tokyo)
	case "ap-northeast-1", "ap-northeast-2", "ap-northeast-3":
		return "https://prod-registry-k8s-io-ap-northeast-1.s3.dualstack.ap-northeast-1.amazonaws.com"
	// Asia Pacific (Singapore)
	case "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4", "ap-southeast-5", "ap-southeast-6", "ap-southeast-7", "ap-east-1", "ap-east-2", "cn-northwest-1", "cn-north-1":
		return "https://prod-registry-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com"
	// Europe (Frankfurt)
	case "eu-central-1", "eu-central-2", "eu-south-1", "eu-south-2", "il-central-1":
		return "https://prod-registry-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com"
	// Europe (Ireland)
	case "eu-west-1", "af-south-1", "eu-west-2", "eu-west-3", "eu-north-1":
		return "https://prod-registry-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com"
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
