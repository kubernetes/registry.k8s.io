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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"k8s.io/registry.k8s.io/cmd/archeio/app"
	"k8s.io/registry.k8s.io/cmd/redirectserver/pkg/mockgcs"
	"k8s.io/registry.k8s.io/cmd/redirectserver/pkg/mocks3"
)

type Harness struct {
	Server *Server
	Config MirrorConfig
}

func NewHarness(t *testing.T) *Harness {
	cfg := MirrorConfig{
		CanonicalLocation: GCSKey{
			Bucket:    "canonical-bucket",
			ObjectKey: "",
		},
		CanonicalFallback: "https://artifacts-fallback.k8s.io",
		InfoURL:           "https://github.com/kubernetes/k8s.io/tree/main/artifacts.k8s.io",
		PrivacyURL:        "https://www.linuxfoundation.org/privacy-policy/",
	}
	mockGCS := mockgcs.New()
	bucket := mockGCS.AddBucket(cfg.CanonicalLocation.Bucket)
	bucket.AddObject("binaries/1.2.3/darwin/arm64/evolution", mockgcs.Object{})
	bucket.AddObject("binaries/1.2.3/darwin/arm64/evolution.sha256", mockgcs.Object{
		Contents: []byte("536241c501f4f77c17c999ae822f4bbc28f724d190a68315bad07aac9ad1229c"),
	})
	bucket.AddObject("binaries/kube.gpg", mockgcs.Object{
		Contents: []byte("k8s is the key"),
	})
	bucket.AddObject("binaries/trigger-500.sha256", mockgcs.Object{
		Override: func(response *http.Response) {
			response.StatusCode = http.StatusInternalServerError
			response.Status = http.StatusText(response.StatusCode)
		},
	})

	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx, option.WithHTTPClient(mockGCS.HTTPClient()))
	if err != nil {
		t.Fatalf("error from storage.NewClient: %v", err)
	}

	hosts := []string{
		"prod-artifacts-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com",
		"prod-artifacts-k8s-io-ap-southeast-1.s3.dualstack.ap-southeast-1.amazonaws.com",
		"prod-artifacts-k8s-io-eu-central-1.s3.dualstack.eu-central-1.amazonaws.com",
		"prod-artifacts-k8s-io-eu-west-1.s3.dualstack.eu-west-1.amazonaws.com",
		"prod-artifacts-k8s-io-us-east-1.s3.dualstack.us-east-2.amazonaws.com",
		"prod-artifacts-k8s-io-us-east-2.s3.dualstack.us-east-2.amazonaws.com",
		"prod-artifacts-k8s-io-us-west-1.s3.dualstack.us-west-1.amazonaws.com",
		"prod-artifacts-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com",
		"prod-artifacts-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com",
	}
	s3 := mocks3.New()
	for _, host := range hosts {
		bucket := s3.AddBucket(host)
		bucket.AddObject("binaries/1.2.3/darwin/arm64/evolution", mocks3.Object{})
	}

	mirrorCache := app.NewCachedBlobChecker(s3.HTTPClient())
	server := NewServer(cfg, storageClient, mirrorCache)
	h := &Harness{}
	h.Config = cfg
	h.Server = server
	return h
}

type TestRequest struct {
	Name            string
	Request         *http.Request
	ExpectedStatus  int
	ExpectedURL     string
	ExpectedContent string
}

func (tc *TestRequest) Run(t *testing.T, harness *Harness) {
	recorder := httptest.NewRecorder()
	harness.Server.ServeHTTP(recorder, tc.Request)
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("error reading body: %v", err)
	}
	want := tc.ExpectedContent
	if want == "" && tc.ExpectedStatus == http.StatusTemporaryRedirect && tc.Request.Method != "HEAD" {
		want = expectedContentForRedirect(tc.ExpectedURL)
	}
	got := strings.TrimSpace(string(body))
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("content did not match expected; got %q, want %q\ndiff=%v", got, want, diff)
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
}

func TestHTTPRequest(t *testing.T) {
	h := NewHarness(t)

	testCases := []TestRequest{
		{
			Name:           "/",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    h.Config.InfoURL,
		},
		{
			Name:           "/privacy",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/privacy", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    h.Config.PrivacyURL,
		},
		{
			Name:            "/not-binaries/",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/v3/", nil),
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: "404 page not found",
		},
		{
			Name:            "/binaries",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries", nil),
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: "404 page not found",
		},
		{
			Name:           "/binaries/",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/binaries/", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://artifacts-fallback.k8s.io/binaries/",
		},
		{
			Name:           "HEAD /binaries/",
			Request:        httptest.NewRequest("HEAD", "http://localhost:8080/binaries/", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://artifacts-fallback.k8s.io/binaries/",
		},
		{
			Name:            "POST /binaries/",
			Request:         httptest.NewRequest("POST", "http://localhost:8080/binaries/", nil),
			ExpectedStatus:  http.StatusMethodNotAllowed,
			ExpectedContent: "Only GET and HEAD are allowed.",
		},
		{
			Name:           "/binaries/1.2.3/darwin/arm64/evolution",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://artifacts-fallback.k8s.io/binaries/1.2.3/darwin/arm64/evolution",
		},
		{
			Name:            "/binaries/not-a-file.sha256",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries/not-a-file.sha256", nil),
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: "Not Found",
		},
		{
			Name:            "/binaries/trigger-500.sha256",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries/trigger-500.sha256", nil),
			ExpectedStatus:  http.StatusInternalServerError,
			ExpectedContent: "Internal Server Error",
		},
		{
			Name:            "/binaries/1.2.3/darwin/arm64/evolution.sha256",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution.sha256", nil),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: "536241c501f4f77c17c999ae822f4bbc28f724d190a68315bad07aac9ad1229c",
		},
		{
			Name:            "/binaries/kube.gpg",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries/kube.gpg", nil),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: "k8s is the key",
		},
		{
			Name: "AWS IP, /binaries/1.2.3/darwin/arm64/evolution",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://prod-artifacts-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com/binaries/1.2.3/darwin/arm64/evolution",
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tc.Run(t, h)
		})
	}
}

// expectedContentForRedirect is the body for a redirect response.
func expectedContentForRedirect(url string) string {
	return fmt.Sprintf("<a href=%q>Temporary Redirect</a>.", url)
}

func TestBinaryRedirection(t *testing.T) {
	h := NewHarness(t)

	testCases := []TestRequest{
		{
			Name:           "/binaries/1.2.3/darwin/arm64/evolution",
			Request:        httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution", nil),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://artifacts-fallback.k8s.io/binaries/1.2.3/darwin/arm64/evolution",
		},
		{
			Name: "Somehow bogus remote addr, /binaries/1.2.3/darwin/arm64/evolution",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution", nil)
				r.RemoteAddr = "35.180.1.1asdfasdfsd:888"
				return r
			}(),
			// NOTE: this one really shouldn't happen, but we want full test coverage
			// This should only happen with a bug in the stdlib http server ...
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: `ParseAddr("35.180.1.1asdfasdfsd"): unexpected character (at "asdfasdfsd")`,
		},
		{
			Name: "AWS IP, exists in mirror",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:80800/binaries/1.2.3/darwin/arm64/evolution", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://prod-artifacts-k8s-io-eu-west-2.s3.dualstack.eu-west-2.amazonaws.com/binaries/1.2.3/darwin/arm64/evolution",
		},
		{
			Name:            "AWS IP, sha file",
			Request:         httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/evolution.sha256", nil),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: "536241c501f4f77c17c999ae822f4bbc28f724d190a68315bad07aac9ad1229c",
		},
		{
			Name: "AWS IP, not in mirror",
			Request: func() *http.Request {
				r := httptest.NewRequest("GET", "http://localhost:8080/binaries/1.2.3/darwin/arm64/something-else", nil)
				r.RemoteAddr = "35.180.1.1:888"
				return r
			}(),
			ExpectedStatus: http.StatusTemporaryRedirect,
			ExpectedURL:    "https://artifacts-fallback.k8s.io/binaries/1.2.3/darwin/arm64/something-else",
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tc.Run(t, h)
		})
	}
}
