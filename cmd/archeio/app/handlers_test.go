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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeHandler(t *testing.T) {
	registryConfig := RegistryConfig{
		// the v2 test below tests being redirected to k8s.gcr.io as that one doesn't have UpstreamRegistryPath
		UpstreamRegistryEndpoint: "https://us.gcr.io",
		UpstreamRegistryPath:     "k8s-artifacts-prod",
		InfoURL:                  "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io",
		PrivacyURL:               "https://www.linuxfoundation.org/privacy-policy/",
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
			ExpectedURL:    "https://us.gcr.io/v2/k8s-artifacts-prod/pause/manifests/latest",
		},
		{
			Name:           "/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us.gcr.io/v2/k8s-artifacts-prod/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "AWS IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us.gcr.io/v2/k8s-artifacts-prod/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "GCP IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.220.26.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us.gcr.io/v2/k8s-artifacts-prod/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
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

func (f *fakeBlobsChecker) BlobExists(blobURL, bucket, hashKey string) bool {
	return f.knownURLs[blobURL]
}

func TestMakeV2Handler(t *testing.T) {
	registryConfig := RegistryConfig{
		UpstreamRegistryEndpoint: "https://k8s.gcr.io",
		UpstreamRegistryPath:     "",
		InfoURL:                  "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io",
		PrivacyURL:               "https://www.linuxfoundation.org/privacy-policy/",
	}
	blobs := fakeBlobsChecker{
		knownURLs: map[string]bool{
			"https://prod-registry-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":         true,
			"https://prod-registry-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e": true,
			"https://prod-registry-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":     true,
			"https://prod-registry-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-east-1.s3.dualstack.us-east-2.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
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
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "Somehow bogus remote addr, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
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
			Name: "AWS IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://prod-registry-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com/containers/images/sha256%3Ada86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name:           "AWS IP, /v2/pause/manifests/latest",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/manifests/latest",
		},
		{
			Name: "AWS IP, /v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567",
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			handler(recorder, tc.Request)
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
