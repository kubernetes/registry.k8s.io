/*
Copyright 2026 The Kubernetes Authors.

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
	"net/http/httptest"
	"testing"
)

// TestTrackPullEventNonImagePath ensures non-image paths emit no event
func TestTrackPullEventNonImagePath(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "http://localhost:8080/v2/weird", nil)
	trackPullEvent(r, "4bf92f3577b34da6a3ce929d0e0e4736", "upstream", nil)
}

func TestParsePullEvent(t *testing.T) {
	testCases := []struct {
		Name              string
		Path              string
		ExpectedImage     string
		ExpectedKind      string
		ExpectedReference string
		ExpectOK          bool
	}{
		{
			Name:              "manifest by tag",
			Path:              "/v2/pause/manifests/latest",
			ExpectedImage:     "pause",
			ExpectedKind:      "manifest",
			ExpectedReference: "latest",
			ExpectOK:          true,
		},
		{
			Name:              "manifest for nested image by digest",
			Path:              "/v2/kubernetes/pause/manifests/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectedImage:     "kubernetes/pause",
			ExpectedKind:      "manifest",
			ExpectedReference: "sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectOK:          true,
		},
		{
			Name:              "blob",
			Path:              "/v2/pause/blobs/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectedImage:     "pause",
			ExpectedKind:      "blob",
			ExpectedReference: "sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e",
			ExpectOK:          true,
		},
		{
			Name:              "tags list",
			Path:              "/v2/pause/tags/list",
			ExpectedImage:     "pause",
			ExpectedKind:      "tag",
			ExpectedReference: "list",
			ExpectOK:          true,
		},
		{
			Name:     "v2 check",
			Path:     "/v2/",
			ExpectOK: false,
		},
		{
			Name:     "catalog",
			Path:     "/v2/_catalog",
			ExpectOK: false,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			image, kind, reference, ok := parsePullEvent(tc.Path)
			if ok != tc.ExpectOK {
				t.Fatalf("expected ok: %v, got: %v", tc.ExpectOK, ok)
			}
			if !ok {
				return
			}
			if image != tc.ExpectedImage {
				t.Fatalf("expected image: %q, got: %q", tc.ExpectedImage, image)
			}
			if kind != tc.ExpectedKind {
				t.Fatalf("expected kind: %q, got: %q", tc.ExpectedKind, kind)
			}
			if reference != tc.ExpectedReference {
				t.Fatalf("expected reference: %q, got: %q", tc.ExpectedReference, reference)
			}
		})
	}
}
