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
	"net/http"
	"os"
	"regexp"

	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

// reImagePath matches image requests per the OCI distribution spec and
// captures the image name, the request kind, and the reference (tag/digest)
// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pull
var reImagePath = regexp.MustCompile(`^/v2/(.+)/(manifests|blobs|tags)/([^/]+)$`)

// parsePullEvent parses a registry API path into analytics event fields,
// returning ok=false for paths that are not image requests
func parsePullEvent(path string) (image, kind, reference string, ok bool) {
	m := reImagePath.FindStringSubmatch(path)
	if m == nil {
		return "", "", "", false
	}
	// normalize to singular event kinds
	switch m[2] {
	case "manifests":
		kind = "manifest"
	case "blobs":
		kind = "blob"
	case "tags":
		kind = "tag"
	}
	return m[1], kind, m[3], true
}

// trackPullEvent emits one structured "image_pull" log event per image
// request, the analytics analog of an Analytics Engine data point.
//
// Events are routed by log sink to Log Analytics / BigQuery for querying,
// see docs/request-handling.md
//
// NOTE: intentionally no client IP or other PII, only the derived
// cloud/region info, per the project privacy policy
func trackPullEvent(r *http.Request, traceID, backend string, ipInfo *cloudcidrs.IPInfo) {
	image, kind, reference, ok := parsePullEvent(r.URL.Path)
	if !ok {
		return
	}
	cloud, region := "external", "external"
	if ipInfo != nil {
		cloud, region = ipInfo.Cloud, ipInfo.Region
	}
	klog.V(1).InfoS("image_pull",
		"image", image,
		"type", kind,
		"reference", reference,
		"userAgent", r.UserAgent(),
		"cloud", cloud,
		"region", region,
		"backend", backend,
		"traceID", traceID,
		"service", os.Getenv("K_SERVICE"),
	)
}
