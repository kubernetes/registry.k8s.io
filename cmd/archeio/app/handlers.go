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
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/oci-proxy/pkg/net/cidrs/aws"
)

const (
	// hard coded for now
	infoURL = "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io"
)

// MakeHandler returns the root archeio HTTP handler
//
// upstream registry should be the url to the primary registry
// archeio is fronting.
func MakeHandler(upstreamRegistry string) http.Handler {
	doV2 := makeV2Handler(upstreamRegistry)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// all valid registry requests should be at /v2/
		// v1 API is super old and not supported by GCR anymore.
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/v2/"):
			doV2(w, r)
		case path == "/":
			http.Redirect(w, r, infoURL, http.StatusPermanentRedirect)
		default:
			klog.V(2).InfoS("unknown request", "path", path)
			http.NotFound(w, r)
		}
	})
}

func makeV2Handler(upstreamRegistry string) func(w http.ResponseWriter, r *http.Request) {
	// matches blob requests, captures the requested blob hash
	reBlob := regexp.MustCompile("^/v2/.*/blobs/sha256:([0-9a-f]{64})$")
	// initialize map of clientIP to AWS region
	regionMapper := aws.NewAWSRegionMapper()
	// capture these in a http handler lambda
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// check if blob request
		matches := reBlob.FindStringSubmatch(path)
		if len(matches) != 2 {
			// doesn't match so just forward it to the main upstream registry
			klog.V(2).InfoS("redirecting non-blob request to upstream registry", "path", path)
			http.Redirect(w, r, upstreamRegistry+path, http.StatusPermanentRedirect)
			return
		}

		// for matches, identify the appropriate backend
		clientIP, err := getClientIP(r)
		if err != nil {
			klog.ErrorS(err, "failed to get client IP")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: revert this!
		// we just need some real data to debug
		klog.InfoS("blob request client info", "clientIP", clientIP, "headers", r.Header, "path", path)

		region, matched := regionMapper.GetIP(clientIP)
		if !matched {
			// no region match, redirect to main upstream registry
			klog.V(2).InfoS("redirecting blob request to upstream registry", "path", path)
			http.Redirect(w, r, upstreamRegistry+path, http.StatusPermanentRedirect)
			return
		}

		bucket := regionToBucket(region)
		hash := matches[1]
		// blobs are in the buckets are stored at /containers/images/sha256:$hash
		// this matches the GCS bucket backing GCR
		klog.V(2).InfoS("redirecting blob request to AWS", "region", region, "path", path)
		http.Redirect(w, r, bucket+"/containers/images/sha256%3A"+hash, http.StatusPermanentRedirect)
	}
}
