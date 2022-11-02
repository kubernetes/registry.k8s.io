//go:build !nointegration
// +build !nointegration

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
	"testing"
)

func TestCachedBlobChecker(t *testing.T) {
	bucket := awsRegionToS3URL("us-east-1")
	blobs := NewCachedBlobChecker(http.DefaultClient)
	testCases := []struct {
		Name         string
		BlobURL      string
		Bucket       string
		HashKey      string
		ExpectExists bool
	}{
		{
			Name:         "known bucket entry",
			BlobURL:      bucket + "/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Bucket:       bucket,
			HashKey:      "3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectExists: true,
		},
		{
			Name:         "known bucket, bad entry",
			Bucket:       bucket,
			BlobURL:      bucket + "/c0ntainers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectExists: false,
		},
		{
			Name:         "bogus bucket on domain without webserver",
			Bucket:       "http://bogus.k8s.io/",
			BlobURL:      "http://bogus.k8s.io/foo",
			HashKey:      "b0guS",
			ExpectExists: false,
		},
	}
	// run test cases in parallel and then serial
	// this populates the cache on the first run while doing parallel testing
	// and allows us to check cached behavior on the second run
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			url := tc.BlobURL
			exists := blobs.BlobExists(url, tc.Bucket, tc.HashKey)
			if exists != tc.ExpectExists {
				t.Fatalf("expected: %v but got: %v", tc.ExpectExists, exists)
			}
		})
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			url := tc.BlobURL
			exists := blobs.BlobExists(url, tc.Bucket, tc.HashKey)
			if exists != tc.ExpectExists {
				t.Fatalf("expected: %v but got: %v", tc.ExpectExists, exists)
			}
		})
	}

}
