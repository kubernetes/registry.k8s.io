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
	"context"
	"net/http"
	"sync"
	"time"

	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"k8s.io/klog/v2"
)

// TODO: replace with a more dynamic way to get the bucket URL
var knownS3Buckets = map[string]string{
	"us-east-1":      "https://adel-us-east-1.s3.dualstack.us-east-1.amazonaws.com",
	"us-east-2":      "https://adel-us-east-2.s3.dualstack.us-east-2.amazonaws.com",
	"us-west-1":      "https://adel-us-west-1.s3.dualstack.us-west-1.amazonaws.com",
	"ap-southeast-1": "https://adel-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com",
	"eu-central-1":   "https://adel-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com",
}

// awsRegionToHostURL returns the base S3 bucket URL for an OCI layer blob given the AWS region
//
// blobs in the buckets should be stored at /containers/images/sha256:$hash
func awsRegionToHostURL(region, defaultURL string) string {
	if url, ok := knownS3Buckets[region]; ok {
		return url
	}
	return defaultURL
}

// blobChecker are used to check if a blob exists, possibly with caching
type blobChecker interface {
	// BlobExistsWithContext should check that blobURL exists
	// bucket and layerHash may be used for caching purposes
	BlobExistsWithContext(ctx context.Context, blobURL string) bool
}

// cachedBlobChecker just performs an HTTP HEAD check against the blob
//
// TODO: potentially replace with a caching implementation
// should be plenty fast for now, HTTP HEAD on s3 is cheap
type cachedBlobChecker struct {
	blobCache
	client *http.Client
}

func newCachedBlobChecker() *cachedBlobChecker {
	// Create and wrap the HTTP client once at initialization
	// This ensures the tracer integration happens when the tracer is ready
	client := &http.Client{
		// Increased timeout to 10s for S3 HEAD requests
		Timeout: time.Second * 10,
	}
	// Wrap with httptrace to instrument the HTTP request
	httptrace.WrapClient(client)

	return &cachedBlobChecker{
		client: client,
	}
}

// cacheEntry stores the result and expiration time
type cacheEntry struct {
	exists    bool
	expiresAt time.Time
}

type blobCache struct {
	m sync.Map
}

// Get returns (exists, found) where found indicates if the URL is in cache
func (b *blobCache) Get(blobURL string) (bool, bool) {
	val, ok := b.m.Load(blobURL)
	if !ok {
		return false, false
	}
	entry := val.(cacheEntry)
	// Check if entry has expired
	if time.Now().After(entry.expiresAt) {
		b.m.Delete(blobURL)
		return false, false
	}
	return entry.exists, true
}

// Put stores the result with a TTL
func (b *blobCache) Put(blobURL string, exists bool) {
	var ttl time.Duration
	ttl = 5 * time.Minute
	entry := cacheEntry{
		exists:    exists,
		expiresAt: time.Now().Add(ttl),
	}
	b.m.Store(blobURL, entry)
}

func (c *cachedBlobChecker) BlobExistsWithContext(ctx context.Context, blobURL string) bool {
	// Check cache first
	if exists, found := c.blobCache.Get(blobURL); found {
		klog.V(3).InfoS("blob existence cache hit", "url", blobURL, "exists", exists)
		// Cache hit - no need for a span, very fast operation
		return exists
	}

	klog.V(3).InfoS("blob existence cache miss", "url", blobURL)

	// Perform HTTP HEAD (this will create its own span)
	exists := c.performHeadCheck(ctx, blobURL)

	// Cache the result (both positive and negative)
	c.blobCache.Put(blobURL, exists)

	return exists
}

func (c *cachedBlobChecker) performHeadCheck(ctx context.Context, blobURL string) bool {
	// Use the pre-wrapped client
	req, err := http.NewRequestWithContext(ctx, "HEAD", blobURL, nil)
	if err != nil {
		klog.Errorf("failed to create HEAD request for %s: %v", blobURL, err)
		return false
	}

	startTime := time.Now()
	r, err := c.client.Do(req)
	duration := time.Since(startTime)

	// fallback to assuming blob is unavailable on errors
	if err != nil {
		klog.Errorf("failed to HEAD %s (took %v): %v", blobURL, duration, err)
		return false
	}
	defer r.Body.Close()

	// if the blob exists it HEAD should return 200 OK
	// this is true for S3 and for OCI registries
	if r.StatusCode == http.StatusOK {
		klog.V(3).InfoS("blob exists", "url", blobURL, "duration", duration)
		return true
	}

	klog.V(3).InfoS("blob does not exist", "url", blobURL, "status", r.StatusCode, "duration", duration)
	return false
}
