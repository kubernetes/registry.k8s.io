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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

func TestMakeHandler(t *testing.T) {
	registryConfig := RegistryConfig{
		UpstreamUsGAR:   Registry{Endpoint: "https://gcr.io", Namespace: "datadoghq"},
		UpstreamEuGAR:   Registry{Endpoint: "https://eu.gcr.io", Namespace: "datadoghq"},
		UpstreamAsiaGAR: Registry{Endpoint: "https://asia.gcr.io", Namespace: "datadoghq"},
		UpstreamACR:     Registry{Endpoint: "https://datadoghq.azurecr.io"},
		UpstreamCDN:     Registry{Endpoint: "https://d3o2h7i3xf2t1t.cloudfront.net"},
		InfoURL:         "https://docs.datadoghq.com/",
		PrivacyURL:      "https://www.datadoghq.com/legal/privacy/",
	}

	handler := MakeHandler(registryConfig)
	testCases := []struct {
		Name           string
		Request        *http.Request
		ExpectedStatus int
		ExpectedURL    string
	}{
		{
			Name:           "/",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    registryConfig.InfoURL,
		},
		{
			Name:           "/privacy",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/privacy", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    registryConfig.PrivacyURL,
		},
		{
			Name:           "/v3/",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v3/", nil),
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "/v2/",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/", nil),
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "/v2",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2", nil),
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "/v2",
			Request:        httptest.NewRequest("HEAD", "http://localhost:8080/v2", nil),
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "/v2/",
			Request:        httptest.NewRequest("POST", "http://localhost:8080/v2/", nil),
			ExpectedStatus: http.StatusMethodNotAllowed,
		},
		{
			Name:           "/v2/pause/manifests/latest",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/manifests/latest",
		},
		{
			Name:           "/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, tc.Request)
			response := recorder.Result()
			if response == nil {
				t.Fatalf("nil response")
			}
			if response.StatusCode != tc.ExpectedStatus {
				t.Fatalf(
					"expected status: %v, but got status: %v",
					http.StatusText(tc.ExpectedStatus),
					http.StatusText(response.StatusCode),
				)
			}
			location, err := response.Location()
			if err != nil {
				if !errors.Is(err, http.ErrNoLocation) {
					t.Fatalf("failed to get response location with error: %v", err)
				} else if tc.ExpectedURL != "" {
					t.Fatalf("expected url: %q but no location was available", tc.ExpectedURL)
				}
			} else if location.String() != tc.ExpectedURL {
				t.Fatalf(
					"expected url: %q, but got: %q",
					tc.ExpectedURL,
					location,
				)
			}
		})
	}
}

type fakeBlobsChecker struct {
	knownURLs map[string]bool
}

func (f *fakeBlobsChecker) BlobExistsWithContext(ctx context.Context, blobURL string) bool {
	return f.knownURLs[blobURL]
}

// fakeIPMapper implements a test version of IPMapper with predetermined IP mappings
type fakeIPMapper struct {
	ipMap map[string]cloudcidrs.IPInfo
}

func (f *fakeIPMapper) GetIP(ip netip.Addr) (cloudcidrs.IPInfo, bool) {
	info, ok := f.ipMap[ip.String()]
	return info, ok
}

func TestMakeV2Handler(t *testing.T) {
	// Build new registry config matching updated code
	registryConfig := RegistryConfig{
		UpstreamUsGAR:   Registry{Endpoint: "https://gcr.io", Namespace: "datadoghq"},
		UpstreamEuGAR:   Registry{Endpoint: "https://eu.gcr.io", Namespace: "datadoghq"},
		UpstreamAsiaGAR: Registry{Endpoint: "https://asia.gcr.io", Namespace: "datadoghq"},
		UpstreamACR:     Registry{Endpoint: "https://datadoghq.azurecr.io"},
		UpstreamCDN:     Registry{Endpoint: "https://d3o2h7i3xf2t1t.cloudfront.net"},
		InfoURL:         "https://docs.datadoghq.com/",
		PrivacyURL:      "https://www.datadoghq.com/legal/privacy/",
	}

	// Initialize the GCP region trie
	gcpRegionTrie = newRegionTrie(registryConfig)

	// Override regionMapper with a test version that has known test IPs
	testMapper := &fakeIPMapper{
		ipMap: map[string]cloudcidrs.IPInfo{
			"10.0.0.1": {Cloud: cloudcidrs.AWS, Region: "us-east-1"},
			"10.0.0.2": {Cloud: cloudcidrs.GCP, Region: "us-central1"},
			"10.0.0.4": {Cloud: cloudcidrs.GCP, Region: "europe-west1"},
		},
	}
	regionMapper = testMapper

	blobs := fakeBlobsChecker{
		knownURLs: map[string]bool{
			"https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e": true,
		},
	}
	handler := makeV2Handler(registryConfig, &blobs)
	testCases := []struct {
		Name           string
		Request        *http.Request
		ExpectedStatus int
		ExpectedURL    string
	}{
		{
			Name:           "/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			// future-proofing tests for other digest algorithms, even though we only have sha256 content as of March 2023
			Name:           "/v2/pause/blobs/sha512:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7",
		},
		{
			Name: "Bogus remote addr -> 400",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.180.1.1asdfasdfsd:888"
				return r
			}(),
			// NOTE: this one really shouldn't happen, but we want full test coverage
			// This should only happen with a bug in the stdlib http server ...
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name: "/v2/_catalog",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/_catalog", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name: "AWS IP, Blob present in S3 -> S3",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "10.0.0.1:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "AWS IP, Manifest -> CDN",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil)
				r.RemoteAddr = "10.0.0.1:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/manifests/latest",
		},
		{
			Name: "Blob not present in S3 -> fallback to CDN",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567", nil)
				r.RemoteAddr = "10.0.0.1:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567",
		},
		{
			Name: "Manifest not present in S3 -> fallback to CDN",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/aaaaa", nil)
				r.RemoteAddr = "10.0.0.1:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://d3o2h7i3xf2t1t.cloudfront.net/v2/pause/manifests/aaaaa",
		},
		{
			Name: "GCP EU -> EU GCR",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil)
				// Use test IP mapped to GCP europe-west1
				r.RemoteAddr = "10.0.0.4:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://eu.gcr.io/v2/datadoghq/pause/manifests/latest",
		},
		{
			Name: "GCP US blob -> US GCR",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "10.0.0.2:1234"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://gcr.io/v2/datadoghq/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			handler(recorder, tc.Request, nil, "", "")
			response := recorder.Result()
			if response == nil {
				t.Fatalf("nil response")
			}
			if response.StatusCode != tc.ExpectedStatus {
				t.Fatalf(
					"expected status: %v, but got status: %v",
					http.StatusText(tc.ExpectedStatus),
					http.StatusText(response.StatusCode),
				)
			}
			location, err := response.Location()
			if err != nil {
				if !errors.Is(err, http.ErrNoLocation) {
					t.Fatalf("failed to get response location with error: %v", err)
				} else if tc.ExpectedURL != "" {
					t.Fatalf("expected url: %q but no location was available", tc.ExpectedURL)
				}
			} else if location.String() != tc.ExpectedURL {
				t.Fatalf(
					"expected url: %q, but got: %q",
					tc.ExpectedURL,
					location,
				)
			}
		})
	}
}
