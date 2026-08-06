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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMakeHandler(t *testing.T) {
	registryConfig := RegistryConfig{
		// the v2 test below tests being redirected to k8s.gcr.io as that one doesn't have UpstreamRegistryPath
		UpstreamRegistryEndpoint: "https://us-central1-docker.pkg.dev",
		UpstreamRegistryPath:     "k8s-artifacts-prod/images",
		InfoURL:                  "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io",
		PrivacyURL:               "https://www.linuxfoundation.org/privacy-policy/",
		// SignatureUpstreamEndpoint intentionally unset to test fallback behavior
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
			Name:           "/v2/ without token gets challenge",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/", nil),
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "/v2 without token gets challenge",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2", nil),
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name:           "HEAD /v2 without token gets challenge",
			Request:        httptest.NewRequest("HEAD", "http://localhost:8080/v2", nil),
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			Name: "/v2/ with correlation token",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/", nil)
				r.Header.Set("Authorization", "Bearer 4bf92f3577b34da6a3ce929d0e0e4736")
				return r
			}(),
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "/token issues correlation token",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/token?service=localhost:8080&scope=repository:pause:pull", nil),
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
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/manifests/latest",
		},
		{
			Name:           "/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "AWS eu-west-3 IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://prod-registry-k8s-io-eu-west-3.s3.dualstack.eu-west-3.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name: "GCP IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.220.26.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			// Without SignatureUpstreamEndpoint, .sig tags fall through to the regional upstream
			Name:           "Cosign .sig tag without canonical upstream falls back to regional",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig",
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
			} else if got := stripTraceID(t, location); got != tc.ExpectedURL {
				t.Fatalf(
					"expected url: %q, but got: %q",
					tc.ExpectedURL,
					got,
				)
			}
		})
	}
}

// stripTraceID removes the trace ID query parameter from a redirect location,
// validating it if present, and returns the remaining URL string
func stripTraceID(t *testing.T, location *url.URL) string {
	t.Helper()
	q := location.Query()
	if traceID := q.Get(traceIDQueryParam); traceID != "" {
		if !reTraceID.MatchString(traceID) {
			t.Fatalf("invalid trace ID in redirect location: %q", traceID)
		}
		q.Del(traceIDQueryParam)
		location.RawQuery = q.Encode()
	}
	return location.String()
}

type fakeBlobsChecker struct {
	knownURLs map[string]bool
}

// TestTraceIDGenerationFailure covers the crypto/rand failure paths
// NOTE: mutates the injectable randRead, must not run in parallel
func TestTraceIDGenerationFailure(t *testing.T) {
	originalRandRead := randRead
	randRead = func([]byte) (int, error) {
		return 0, errors.New("entropy exhausted")
	}
	defer func() { randRead = originalRandRead }()

	if id, err := GenerateTraceID(); err == nil || id != "" {
		t.Fatalf("expected error and empty ID, got: %q, %v", id, err)
	}

	// traceIDForRequest falls back to empty on generation failure
	r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil)
	if got := traceIDForRequest(r); got != "" {
		t.Fatalf("expected empty trace ID, got: %q", got)
	}

	// serveToken cannot issue a token without an ID
	recorder := httptest.NewRecorder()
	serveToken(recorder, httptest.NewRequest("GET", "http://localhost:8080/token", nil))
	if status := recorder.Result().StatusCode; status != http.StatusInternalServerError {
		t.Fatalf("expected 500 from /token, got: %v", status)
	}
}

func TestWithTraceID(t *testing.T) {
	t.Parallel()
	// empty trace ID is a no-op
	if got := withTraceID("https://example.com/foo", ""); got != "https://example.com/foo" {
		t.Fatalf("unexpected url: %q", got)
	}
	// existing query strings are appended to
	expected := "https://example.com/foo?a=b&" + traceIDQueryParam + "=4bf92f3577b34da6a3ce929d0e0e4736"
	if got := withTraceID("https://example.com/foo?a=b", "4bf92f3577b34da6a3ce929d0e0e4736"); got != expected {
		t.Fatalf("expected url: %q, got: %q", expected, got)
	}
}

func TestRequestScheme(t *testing.T) {
	t.Parallel()
	// plain local request
	r := httptest.NewRequest("GET", "http://localhost:8080/v2/", nil)
	if got := requestScheme(r); got != "http" {
		t.Fatalf("expected http, got: %q", got)
	}
	// forwarded proto from load balancer wins
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := requestScheme(r); got != "https" {
		t.Fatalf("expected https, got: %q", got)
	}
	// direct TLS
	r = httptest.NewRequest("GET", "https://localhost:8080/v2/", nil)
	if got := requestScheme(r); got != "https" {
		t.Fatalf("expected https, got: %q", got)
	}
}
func (f *fakeBlobsChecker) BlobExists(blobURL, _ string) bool {
	return f.knownURLs[blobURL]
}

func TestMakeV2Handler(t *testing.T) {
	registryConfig := RegistryConfig{
		UpstreamRegistryEndpoint:  "https://k8s.gcr.io",
		UpstreamRegistryPath:      "",
		SignatureUpstreamEndpoint: "https://us-central1-docker.pkg.dev",
		InfoURL:                   "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io",
		PrivacyURL:                "https://www.linuxfoundation.org/privacy-policy/",
	}
	blobs := fakeBlobsChecker{
		knownURLs: map[string]bool{
			"https://prod-registry-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":         true,
			"https://prod-registry-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e": true,
			"https://prod-registry-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":     true,
			"https://prod-registry-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-eu-west-3.s3.dualstack.eu-west-3.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-east-1.s3.dualstack.us-east-2.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
			"https://prod-registry-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e":           true,
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
			// future-proofing tests for other digest algorithms, even though we only have sha256 content as of March 2023
			Name:           "/v2/pause/blobs/sha512:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/blobs/sha256:3b0998121425143be7164ea1555efbdf5b8a02ceedaa26e01910e7d017ff78ddbba27877bd42510a06cc14ac1bc6c451128ca3f0d0afba28b695e29b2702c9c7",
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
			Name: "AWS eu-west-3 IP, /v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://prod-registry-k8s-io-eu-west-3.s3.dualstack.eu-west-3.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
		},
		{
			Name:           "Fetching image manifest, /v2/pause/manifests/latest",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/latest", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/manifests/latest",
		},
		{
			Name: "AWS eu-west-3 IP, /v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/blobs/sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1234567",
		},
		{
			Name:           "Cosign .sig tag redirects to canonical upstream",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig",
		},
		{
			Name:           "Cosign .att tag redirects to canonical upstream",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.att", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.att",
		},
		{
			Name:           "HEAD on .sig tag redirects to canonical upstream",
			Request:        httptest.NewRequest("HEAD", "http://localhost:8080/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig",
		},
		{
			Name:           "Cosign .sig tag for nested image name",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/kubernetes/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://us-central1-docker.pkg.dev/v2/kubernetes/pause/manifests/sha256-da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e.sig",
		},
		{
			Name:           "Tag list still redirects to regional upstream",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/v2/pause/tags/list", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://k8s.gcr.io/v2/pause/tags/list",
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
			} else if got := stripTraceID(t, location); got != tc.ExpectedURL {
				t.Fatalf(
					"expected url: %q, but got: %q",
					tc.ExpectedURL,
					got,
				)
			}
		})
	}
}

func TestTraceIDCorrelation(t *testing.T) {
	registryConfig := RegistryConfig{
		UpstreamRegistryEndpoint: "https://k8s.gcr.io",
	}
	blobs := fakeBlobsChecker{
		knownURLs: map[string]bool{
			// NOTE: existence-cache keys must NOT contain the trace ID
			"https://prod-registry-k8s-io-eu-west-3.s3.dualstack.eu-west-3.amazonaws.com/containers/images/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e": true,
		},
	}
	handler := makeV2Handler(registryConfig, &blobs)

	const wantTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	testCases := []struct {
		Name            string
		Header          string
		HeaderValue     string
		ExpectedTraceID string
	}{
		{
			Name:            "traceparent is honored",
			Header:          "traceparent",
			HeaderValue:     "00-" + wantTraceID + "-00f067aa0ba902b7-01",
			ExpectedTraceID: wantTraceID,
		},
		{
			Name:            "X-Cloud-Trace-Context is honored",
			Header:          "X-Cloud-Trace-Context",
			HeaderValue:     wantTraceID + "/123;o=1",
			ExpectedTraceID: wantTraceID,
		},
		{
			Name:            "Bearer correlation token identifies the pull session",
			Header:          "Authorization",
			HeaderValue:     "Bearer " + wantTraceID,
			ExpectedTraceID: wantTraceID,
		},
		{
			Name:        "malformed Bearer token falls back to generated ID",
			Header:      "Authorization",
			HeaderValue: "Bearer no\"t-a-trace-id",
		},
		{
			Name: "trace ID is generated when no trace headers are sent",
		},
		{
			Name:        "malformed traceparent falls back to generated ID",
			Header:      "traceparent",
			HeaderValue: "00-junk\"><script>-x-01",
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", "http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", nil)
			r.RemoteAddr = "35.180.1.1:888" // AWS eu-west-3 => S3 redirect
			if tc.Header != "" {
				r.Header.Set(tc.Header, tc.HeaderValue)
			}
			recorder := httptest.NewRecorder()
			handler(recorder, r)
			response := recorder.Result()
			if response.StatusCode != http.StatusTemporaryRedirect {
				t.Fatalf("expected redirect, got status: %v", response.StatusCode)
			}
			// the response header and the redirect query param must carry
			// the same valid trace ID
			headerTraceID := response.Header.Get(traceIDHeader)
			if !reTraceID.MatchString(headerTraceID) {
				t.Fatalf("expected valid trace ID in %s header, got: %q", traceIDHeader, headerTraceID)
			}
			location, err := response.Location()
			if err != nil {
				t.Fatalf("failed to get response location: %v", err)
			}
			locationTraceID := location.Query().Get(traceIDQueryParam)
			if locationTraceID != headerTraceID {
				t.Fatalf("trace ID mismatch, header: %q, location: %q", headerTraceID, locationTraceID)
			}
			if tc.ExpectedTraceID != "" && headerTraceID != tc.ExpectedTraceID {
				t.Fatalf("expected trace ID %q, got: %q", tc.ExpectedTraceID, headerTraceID)
			}
		})
	}
}

// TestPullSessionCorrelation simulates a full image pull the way OCI clients
// behave: /v2/ challenge -> token fetch -> manifest + blob requests with the
// Bearer token, asserting one consistent trace ID across the whole session
func TestPullSessionCorrelation(t *testing.T) {
	registryConfig := RegistryConfig{
		UpstreamRegistryEndpoint: "https://k8s.gcr.io",
	}
	handler := MakeHandler(registryConfig)

	do := func(method, target, authorization string) *http.Response {
		r := httptest.NewRequest(method, target, nil)
		r.RemoteAddr = "35.180.1.1:888"
		if authorization != "" {
			r.Header.Set("Authorization", authorization)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result()
	}

	// 1. client probes /v2/ and should be challenged
	response := do("GET", "http://localhost:8080/v2/", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 challenge on /v2/, got: %v", response.StatusCode)
	}
	challenge := response.Header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `realm="http://localhost:8080/token"`) {
		t.Fatalf("unexpected WWW-Authenticate challenge: %q", challenge)
	}

	// 2. client fetches a token from the realm
	response = do("GET", "http://localhost:8080/token?service=localhost:8080&scope=repository:pause:pull", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /token, got: %v", response.StatusCode)
	}
	tokenBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read token response: %v", err)
	}
	tokenResponse := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(tokenBody, &tokenResponse); err != nil {
		t.Fatalf("failed to parse token response %q: %v", string(tokenBody), err)
	}
	pullID := tokenResponse.Token
	if !reTraceID.MatchString(pullID) {
		t.Fatalf("expected valid trace ID token, got: %q", pullID)
	}

	// 3. all subsequent requests with the Bearer token must carry the same ID
	authorization := "Bearer " + pullID
	for _, target := range []string{
		"http://localhost:8080/v2/pause/manifests/latest",
		"http://localhost:8080/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
	} {
		response := do("GET", target, authorization)
		if response.StatusCode != http.StatusTemporaryRedirect {
			t.Fatalf("expected redirect for %q, got: %v", target, response.StatusCode)
		}
		if got := response.Header.Get(traceIDHeader); got != pullID {
			t.Fatalf("expected trace ID %q for %q, got: %q", pullID, target, got)
		}
		location, err := response.Location()
		if err != nil {
			t.Fatalf("failed to get location for %q: %v", target, err)
		}
		if got := location.Query().Get(traceIDQueryParam); got != pullID {
			t.Fatalf("expected trace ID %q in location for %q, got: %q", pullID, target, got)
		}
	}
}
