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
	"strings"
	"testing"

	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

func TestRegionToAWSRegionToHostURL(t *testing.T) {
	// ensure known regions return a configured bucket
	regions := []string{}
	for _, ipInfo := range cloudcidrs.AllIPInfos() {
		// AWS regions, excluding "GLOBAL" meta region, AWS US Gov Cloud and European Soveign Cloud
		if ipInfo.Cloud == cloudcidrs.AWS &&
			ipInfo.Region != "GLOBAL" && !strings.HasPrefix(ipInfo.Region, "us-gov-") && !strings.HasPrefix(ipInfo.Region, "eusc-") {
			regions = append(regions, ipInfo.Region)
		}
	}
	for _, region := range regions {
		url := awsRegionToHostURL(region, "")
		if url == "" {
			t.Fatalf("received empty string for known region %q", region)
		}
	}
	// test default region
	if url := awsRegionToHostURL("nonsensical-region", "____default____"); url != "____default____" {
		t.Fatalf("received non-empty URL string for made up region \"nonsensical-region\": %q", url)
	}
}

func TestBlobCache(t *testing.T) {
	bc := &blobCache{}
	bc.Put("foo")
	if !bc.Get("foo") {
		t.Fatal("Cache did not contain key we just put")
	}
	if bc.Get("bar") {
		t.Fatal("Cache contained key we did not put")
	}
}

func TestHeadStatusAcceptsManifests(t *testing.T) {
	for _, mediaType := range []string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		// only accepted by */*
		"application/vnd.docker.distribution.manifest.v1+prettyjws",
		"application/vnd.oci.artifact.manifest.v1+json",
	} {
		t.Run(mediaType, func(t *testing.T) {
			t.Parallel()
			// like Artifact Registry, answer manifest requests with 404 if
			// the media type of the manifest is not accepted
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead {
					t.Errorf("unexpected method %q", r.Method)
				}
				for accepted := range strings.SplitSeq(r.Header.Get("Accept"), ",") {
					if accepted = strings.TrimSpace(accepted); accepted == mediaType || accepted == "*/*" {
						w.Header().Set("Content-Type", mediaType)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()
			status, err := headStatus(server.URL+"/v2/pause/manifests/sha256:da86e6ba6ca197bf6bc5e9d900febd906b133eaa4750e6bed647b0fbe50ed43e", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status != http.StatusOK {
				t.Fatalf("expected status %d, got: %d", http.StatusOK, status)
			}
		})
	}
}
