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
	"path"
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/pkg/clientip"
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

type RegistryConfig struct {
	UpstreamRegistryEndpoint string
	UpstreamRegistryPath     string
	InfoURL                  string
	PrivacyURL               string
	DefaultAWSBaseURL        string
}

// MakeHandler returns the root archeio HTTP handler
//
// upstream registry should be the url to the primary registry
// archeio is fronting.
//
// Exact behavior should be documented in docs/request-handling.md
func MakeHandler(rc RegistryConfig) http.Handler {
	blobs := newCachedBlobChecker()
	doV2 := makeV2Handler(rc, blobs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only allow GET, HEAD
		// this is all a client needs to pull images
		// we do *not* support mutation
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Only GET and HEAD are allowed.", http.StatusMethodNotAllowed)
			return
		}
		// all valid registry requests should be at /v2/
		// v1 API is super old and not supported by GCR anymore.
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/v2"):
			doV2(w, r)
		case path == "/":
			http.Redirect(w, r, rc.InfoURL, http.StatusTemporaryRedirect)
		case strings.HasPrefix(path, "/privacy"):
			http.Redirect(w, r, rc.PrivacyURL, http.StatusTemporaryRedirect)
		default:
			klog.V(2).InfoS("unknown request", "path", path)
			http.NotFound(w, r)
		}
	})
}

func makeV2Handler(rc RegistryConfig, blobs blobChecker) func(w http.ResponseWriter, r *http.Request) {
	// matches blob requests, captures the requested blob hash
	reBlob := regexp.MustCompile("^/v2/.*/blobs/sha256:([0-9a-f]{64})$")
	// initialize map of clientIP to AWS region
	regionMapper := cloudcidrs.NewIPMapper()
	// capture these in a http handler lambda
	return func(w http.ResponseWriter, r *http.Request) {
		rPath := r.URL.Path

		// we only care about publicly readable GCR as the backing registry
		// or publicly readable blob storage
		//
		// when the client attempts to probe the API for auth, we always return
		// 200 OK so it will not attempt to request an auth token
		//
		// this makes it easier to redirect to backends with different
		// repo namespacing without worrying about incorrect token scope
		//
		// it turns out publicly readable GCR repos do not actually care about
		// the presence of a token for any API calls, despite the /v2/ API call
		// returning 401, prompting token auth
		if rPath == "/v2/" || rPath == "/v2" {
			klog.V(2).InfoS("serving 200 OK for /v2/ check", "path", rPath)
			// NOTE: OCI does not require this, but the docker v2 spec include it, and GCR sets this
			// Docker distribution v2 clients may fallback to an older version if this is not set.
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
			return
		}

		// check if blob request
		matches := reBlob.FindStringSubmatch(rPath)
		if len(matches) != 2 {
			// not a blob request so forward it to the main upstream registry
			redirectURL := upstreamRedirectURL(rc, rPath)
			klog.V(2).InfoS("redirecting manifest request to upstream registry", "path", rPath, "redirect", redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		// it is a blob request, grab the hash for later
		hash := matches[1]

		// for blob requests, check the client IP and determine the best backend
		clientIP, err := clientip.Get(r)
		if err != nil {
			// this should not happen
			klog.ErrorS(err, "failed to get client IP")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// if client is coming from GCP, stay in GCP
		ipInfo, ipIsKnown := regionMapper.GetIP(clientIP)
		if ipIsKnown && ipInfo.Cloud == cloudcidrs.GCP {
			redirectURL := upstreamRedirectURL(rc, rPath)
			klog.V(2).InfoS("redirecting GCP blob request to upstream registry", "path", rPath, "redirect", redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}

		// check if blob is available in our AWS layer storage for the region
		region := ""
		if ipIsKnown {
			region = ipInfo.Region
		}
		bucketURL := awsRegionToHostURL(region, rc.DefaultAWSBaseURL)
		// this matches GCR's GCS layout, which we will use for other buckets
		blobURL := bucketURL + "/containers/images/sha256%3A" + hash
		if blobs.BlobExists(blobURL, bucketURL, hash) {
			// blob known to be available in AWS, redirect client there
			klog.V(2).InfoS("redirecting blob request to AWS", "path", rPath)
			http.Redirect(w, r, blobURL, http.StatusTemporaryRedirect)
			return
		}

		// fall back to redirect to upstream
		redirectURL := upstreamRedirectURL(rc, rPath)
		klog.V(2).InfoS("redirecting blob request to upstream registry", "path", rPath, "redirect", redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	}
}

func upstreamRedirectURL(rc RegistryConfig, originalPath string) string {
	return rc.UpstreamRegistryEndpoint + path.Join("/v2/", rc.UpstreamRegistryPath, strings.TrimPrefix(originalPath, "/v2"))
}
