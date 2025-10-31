//go:build !nointegration

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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"testing"

	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

func TestIntegrationCachedBlobChecker(t *testing.T) {
	t.Parallel()
	bucket := awsRegionToHostURL("us-east-1", "")
	blobs := newCachedBlobChecker()
	testCases := []struct {
		Name         string
		BlobURL      string
		Bucket       string
		HashKey      string
		ExpectExists bool
	}{
		{
			Name:         "known bucket entry",
			BlobURL:      bucket + "/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectExists: true,
		},
		// to cover the case that we get a cache hit
		{
			Name:         "same-known bucket entry",
			BlobURL:      bucket + "/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectExists: true,
		},
		{
			Name:         "known bucket, bad entry",
			BlobURL:      bucket + "/c0ntainers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectExists: false,
		},
		{
			Name:         "bogus bucket on domain without webserver",
			BlobURL:      "http://bogus.k8s.io/foo",
			ExpectExists: false,
		},
	}
	// run test cases in parallel and then serial
	// this populates the cache on the first run while doing parallel testing
	// and allows us to check cached behavior on the second run
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			url := tc.BlobURL
			exists := blobs.BlobExists(url)
			if exists != tc.ExpectExists {
				t.Fatalf("expected: %v but got: %v", tc.ExpectExists, exists)
			}
		})
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			url := tc.BlobURL
			exists := blobs.BlobExists(url)
			if exists != tc.ExpectExists {
				t.Fatalf("expected: %v but got: %v", tc.ExpectExists, exists)
			}
		})
	}
}

func TestIntegrationAllBucketsValid(t *testing.T) {
	t.Parallel()
	// a known pause image blob
	const testBlob = "da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e"
	expectedDigest, err := hex.DecodeString(testBlob)
	if err != nil {
		t.Fatalf("Failed to decode test blob digest: %v", err)
	}
	// iterate all AWS regions and their mapped buckets
	ipInfos := cloudcidrs.AllIPInfos()
	for i := range ipInfos {
		ipInfo := ipInfos[i]
		// we only have bucket mappings for AWS currently
		// otherwise these are the deployed terraform defaults,
		// which are a subset of the buckets for AWS-external traffic
		// see also: https://github.com/kubernetes/registry.k8s.io/issues/194
		if ipInfo.Cloud != cloudcidrs.AWS {
			continue
		}
		// skip regions that aren't mapped and would've used the default
		baseURL := awsRegionToHostURL(ipInfo.Region, "")
		if baseURL == "" {
			continue
		}
		// for all remaining regions, fetch a real blob to make sure this
		// bucket will work
		t.Run(ipInfo.Region, func(t *testing.T) {
			t.Parallel()
			url := baseURL + "/containers/images/sha256:" + testBlob
			// this is test code, the URL is not user supplied
			// nolint:gosec
			r, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to get %q: %v", url, err)
			}
			defer r.Body.Close()
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("Failed to read body for %q: %v", url, err)
			}
			digest := sha256.Sum256(b)
			if !bytes.Equal(digest[:], expectedDigest) {
				t.Fatalf("Digest for %q was %q but expected %q", url, hex.EncodeToString(digest[:]), hex.EncodeToString(expectedDigest))
			}
		})
	}
}
