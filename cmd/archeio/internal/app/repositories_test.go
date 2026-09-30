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
	"net/http/httptest"
	"testing"
)

func TestListUpstreamRepositories(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name        string
		Status      int
		Body        string
		ExpectError bool
		Expected    []string
	}{
		{
			Name:     "child repositories",
			Status:   http.StatusOK,
			Body:     `{"child":["pause","sig-storage"],"tags":[],"manifest":{}}`,
			Expected: []string{"pause", "sig-storage"},
		},
		{
			Name:        "no children",
			Status:      http.StatusOK,
			Body:        `{"child":[]}`,
			ExpectError: true,
		},
		{
			Name:        "invalid json",
			Status:      http.StatusOK,
			Body:        `not json`,
			ExpectError: true,
		},
		{
			Name:        "non-OK status",
			Status:      http.StatusForbidden,
			Body:        `{}`,
			ExpectError: true,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var gotPath string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tc.Status)
				_, _ = w.Write([]byte(tc.Body))
			}))
			defer s.Close()

			repositories, err := ListUpstreamRepositories(RegistryConfig{
				UpstreamRegistryEndpoint: s.URL,
				UpstreamRegistryPath:     "k8s-artifacts-prod/images",
			})
			if tc.ExpectError {
				if err == nil {
					t.Fatalf("expected error but got repositories: %v", repositories)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != "/v2/k8s-artifacts-prod/images/tags/list" {
				t.Fatalf("unexpected list path: %q", gotPath)
			}
			if len(repositories) != len(tc.Expected) {
				t.Fatalf("expected %d repositories but got %v", len(tc.Expected), repositories)
			}
			for _, expected := range tc.Expected {
				if _, ok := repositories[expected]; !ok {
					t.Fatalf("expected repository %q in %v", expected, repositories)
				}
			}
		})
	}
}

func TestListUpstreamRepositoriesUnreachable(t *testing.T) {
	t.Parallel()
	if _, err := ListUpstreamRepositories(RegistryConfig{
		UpstreamRegistryEndpoint: "http://bogus.k8s.io",
	}); err == nil {
		t.Fatal("expected error for unreachable registry")
	}
}

func TestTopLevelRepository(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"/v2/pause/manifests/latest":                  "pause",
		"/v2/sig-storage/csi-provisioner/tags/list":   "sig-storage",
		"/v2/pause/blobs/sha256:da86e6ba6ca197bf6bca": "pause",
		"/v2/pause": "pause",
		"/v2/":      "",
		"/v2":       "",
		"/token":    "",
	}
	for requestPath, expected := range testCases {
		if got := topLevelRepository(requestPath); got != expected {
			t.Fatalf("expected %q for %q but got %q", expected, requestPath, got)
		}
	}
}

func TestV2HandlerUnknownRepository(t *testing.T) {
	t.Parallel()
	registryConfig := RegistryConfig{
		UpstreamRegistryEndpoint: "https://k8s.gcr.io",
	}
	knownRepositories := map[string]struct{}{
		"pause":       {},
		"sig-storage": {},
	}
	handler := makeV2Handler(registryConfig, &fakeBlobsChecker{}, &fakeUpstreamChecker{}, knownRepositories)

	testCases := []struct {
		Name           string
		Target         string
		ExpectedStatus int
	}{
		{
			Name:           "known repository",
			Target:         "http://localhost:8080/v2/pause/manifests/latest",
			ExpectedStatus: http.StatusTemporaryRedirect,
		},
		{
			Name:           "known nested repository",
			Target:         "http://localhost:8080/v2/sig-storage/csi-provisioner/manifests/latest",
			ExpectedStatus: http.StatusTemporaryRedirect,
		},
		{
			Name:           "unknown repository manifest",
			Target:         "http://localhost:8080/v2/does-not-exist/manifests/latest",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			Name:           "unknown repository blob",
			Target:         "http://localhost:8080/v2/does-not-exist/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			// the API version check must still work
			Name:           "/v2/ check",
			Target:         "http://localhost:8080/v2/",
			ExpectedStatus: http.StatusUnauthorized,
		},
		{
			// listing the repositories themselves is not an image request
			Name:           "root tags list",
			Target:         "http://localhost:8080/v2/tags/list",
			ExpectedStatus: http.StatusTemporaryRedirect,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tc.Target, nil)
			r.RemoteAddr = "35.180.1.1:888"
			handler(recorder, r)
			if status := recorder.Result().StatusCode; status != tc.ExpectedStatus {
				t.Fatalf("expected status %d but got %d", tc.ExpectedStatus, status)
			}
		})
	}
}
