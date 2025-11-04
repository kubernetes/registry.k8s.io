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
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/armon/go-radix"
	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/pkg/net/clientip"
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

var regionMapper = cloudcidrs.NewIPMapper()
var gcpRegionTrie *radix.Tree

// regionConfig holds registry and region type information
type regionConfig struct {
	registry   Registry
	regionType string
}

// newRegionTrie creates a new radix tree with GCP region prefixes
func newRegionTrie(rc RegistryConfig) *radix.Tree {
	tree := radix.New()

	// Insert GCP region prefixes
	prefixes := map[string]regionConfig{
		"europe":       {rc.UpstreamEuGAR, "EU"},
		"me-":          {rc.UpstreamEuGAR, "EU"},
		"africa":       {rc.UpstreamEuGAR, "EU"},
		"asia":         {rc.UpstreamAsiaGAR, "Asia"},
		"australia":    {rc.UpstreamAsiaGAR, "Asia"},
		"us-":          {rc.UpstreamUsGAR, "US"},
		"northamerica": {rc.UpstreamUsGAR, "US"},
		"southamerica": {rc.UpstreamUsGAR, "US"},
	}

	for prefix, config := range prefixes {
		tree.Insert(prefix, config)
	}

	return tree
}

// findGCPRegistry looks up a region using radix tree prefix matching
func findGCPRegistry(region string) (*Registry, string) {
	_, value, found := gcpRegionTrie.LongestPrefix(region)
	if !found {
		return nil, ""
	}

	config := value.(regionConfig)
	return &config.registry, config.regionType
}

// statusRecorder wraps http.ResponseWriter to capture the final status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

type Registry struct {
	Endpoint  string
	Namespace string
}

type RegistryConfig struct {
	UpstreamUsGAR   Registry
	UpstreamEuGAR   Registry
	UpstreamAsiaGAR Registry
	UpstreamACR     Registry
	UpstreamCDN     Registry
	InfoURL         string
	PrivacyURL      string
}

// MakeHandler returns the root archeio HTTP handler
//
// upstream registry should be the url to the primary registry
// archeio is fronting.
//
// Exact behavior should be documented in docs/request-handling.md
func MakeHandler(rc RegistryConfig) http.Handler {
	// Initialize the GCP region trie once
	gcpRegionTrie = newRegionTrie(rc)

	blobs := newCachedBlobChecker()
	doV2 := makeV2Handler(rc, blobs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		klog.Infof("Handling request: %s %s", r.Method, r.URL.Path)

		// Initialize statsd client in handler
		statsdHost := getEnv("DD_AGENT_HOST", "localhost")
		statsdPort := getEnv("DD_DOGSTATSD_PORT", "8125")
		statsdClient, err := statsd.New(statsdHost+":"+statsdPort, statsd.WithNamespace(getEnv("DD_SERVICE", "dd-registry")+"."))
		if err != nil {
			klog.V(3).InfoS("Failed to create statsd client, metrics will not be sent", "error", err)
			statsdClient = nil
		} else {
			defer statsdClient.Close()
		}

		// Track response status
		statusRecorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		w = statusRecorder

		// Get client IP info for tags
		clientIP, ipErr := clientip.Get(r)
		var sourceIPCloud, sourceIPRegion string
		if ipErr == nil {
			ipInfo, ipIsKnown := regionMapper.GetIP(clientIP)
			if ipIsKnown {
				sourceIPCloud = ipInfo.Cloud
				sourceIPRegion = ipInfo.Region
			}
		}

		// only allow GET, HEAD
		// this is all a client needs to pull images
		// we do *not* support mutation
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			statusRecorder.statusCode = http.StatusMethodNotAllowed
			http.Error(w, "Only GET and HEAD are allowed.", http.StatusMethodNotAllowed)
			sendMetrics(statsdClient, statusRecorder.statusCode, r.Method, sourceIPCloud, sourceIPRegion, r.URL.Path, time.Since(start))
			return
		}
		// all valid registry requests should be at /v2/
		// v1 API is super old and not supported by GCR anymore.
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/v2"):
			doV2(w, r, statsdClient, sourceIPCloud, sourceIPRegion)
		case path == "/":
			statusRecorder.statusCode = http.StatusTemporaryRedirect
			http.Redirect(w, r, rc.InfoURL, http.StatusTemporaryRedirect)
		case strings.HasPrefix(path, "/privacy"):
			statusRecorder.statusCode = http.StatusTemporaryRedirect
			http.Redirect(w, r, rc.PrivacyURL, http.StatusTemporaryRedirect)
		default:
			statusRecorder.statusCode = http.StatusNotFound
			klog.V(2).InfoS("unknown request", "path", path)
			http.NotFound(w, r)
		}

		// Record metrics
		sendMetrics(statsdClient, statusRecorder.statusCode, r.Method, sourceIPCloud, sourceIPRegion, path, time.Since(start))
	})
}

func sendMetrics(client *statsd.Client, status int, method, sourceIPCloud, sourceIPRegion, targetPath string, duration time.Duration) {
	if client == nil {
		return
	}
	tags := []string{
		"status:" + strconv.Itoa(status),
		"method:" + method,
		"target_path:" + targetPath,
	}
	if sourceIPCloud != "" {
		tags = append(tags, "source_ip_cloud:"+sourceIPCloud)
	}
	if sourceIPRegion != "" {
		tags = append(tags, "source_ip_region:"+sourceIPRegion)
	}
	client.Incr("request.count", tags, 1)
	client.Timing("request.duration", duration, tags, 1)
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func makeV2Handler(rc RegistryConfig, blobs blobChecker) func(w http.ResponseWriter, r *http.Request, statsdClient *statsd.Client, sourceIPCloud, sourceIPRegion string) {
	// matches blob and manifests requests, captures the requested blob hash and the manifest's reference
	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pull
	// Blobs are at `/v2/<name>/blobs/<digest>`
	// Manifests are at `/v2/<name>/manifests/<reference>`
	reBlobOrManifest := regexp.MustCompile("^/v2/.*/(blobs|manifests)/.*$")
	// capture these in a http handler lambda
	return func(w http.ResponseWriter, r *http.Request, statsdClient *statsd.Client, sourceIPCloud, sourceIPRegion string) {
		rPath := r.URL.Path
		// check the client IP and determine the best backend
		// It is also crucial for oauth2 token validation
		clientIP, err := clientip.Get(r)
		if err != nil {
			// this should not happen
			klog.ErrorS(err, "failed to get client IP")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Stay in the same cloud provider
		ipInfo, _ := regionMapper.GetIP(clientIP)

		// when the client attempts to probe the API for auth
		// For Azure, we redirect to the upstream registry to handle the auth token
		// as ACR requires this token.
		// For others, we serve 200 OK as we'll redirect to s3 or cloudfront
		if rPath == "/v2/" || rPath == "/v2" {
			if ipInfo.Cloud == cloudcidrs.AZ {
				redirectURL, err := redirectUpstream(rc, rPath, ipInfo, rc.UpstreamACR)
				if err != nil {
					klog.ErrorS(err, "failed to build redirect URL")
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				klog.V(2).Infof("redirecting oauth request to %s", redirectURL)
				http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				return
			}
			klog.V(2).InfoS("serving 200 OK for /v2/ check", "path", rPath)
			// NOTE: OCI does not require this, but the docker v2 spec include it, and GCR sets this
			// Docker distribution v2 clients may fallback to an older version if this is not set.
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
			return
		}
		// we don't support the non-standard _catalog API
		// https://github.com/kubernetes/registry.k8s.io/issues/162
		if rPath == "/v2/_catalog" {
			http.Error(w, "_catalog is not supported", http.StatusNotFound)
			return
		}

		// If the request is not a blob or manifest request, forward it to an upstream registry (not cdn)
		matches := reBlobOrManifest.MatchString(rPath)
		if !matches {
			klog.V(2).Infof("not a blob or manifest request: %v", rPath)
			redirectURL, err := redirectUpstream(rc, rPath, ipInfo, rc.UpstreamUsGAR)
			if err != nil {
				klog.ErrorS(err, "failed to build redirect URL")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			klog.V(2).Infof("redirecting manifest request to %s", redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}

		// If the request is a blob or manifest request from non-AWS, route to appropriate registry
		if ipInfo.Cloud != cloudcidrs.AWS {
			klog.V(2).Infof("cloud not aws: %v", ipInfo)
			// For GCP/Azure, redirectUpstream will route to the appropriate regional registry
			// For unknown clouds, it will use the default (CDN)
			redirectURL, err := redirectUpstream(rc, rPath, ipInfo, rc.UpstreamCDN)
			if err != nil {
				klog.ErrorS(err, "failed to build redirect URL")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			klog.V(2).Infof("redirecting blob or manifest request to %s", redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}

		// If the request is from AWS, check if the blob is available in our AWS layer storage for the region
		// check if blob is available in our AWS layer storage for the region
		region := ipInfo.Region
		bucketURL := awsRegionToHostURL(region, rc.UpstreamCDN.Endpoint)
		// this matches GCR's GCS layout, which we will use for other buckets
		blobURL, err := url.JoinPath(bucketURL, rPath)
		if err != nil {
			klog.ErrorS(err, "failed to join URL path", "path", rPath, "bucketURL", bucketURL)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		klog.V(2).Infof("Checking if blob exists: %s", blobURL)

		// Use the context-aware version for tracing
		if blobs.BlobExistsWithContext(r.Context(), blobURL) {
			// blob known to be available in AWS, redirect client there
			klog.V(2).Infof("AWS: redirecting blob request to %s", blobURL)
			http.Redirect(w, r, blobURL, http.StatusTemporaryRedirect)
			return
		}

		// fall back to redirect to cdn
		redirectURL, err := redirectUpstream(rc, rPath, ipInfo, rc.UpstreamCDN)
		if err != nil {
			klog.ErrorS(err, "failed to build redirect URL")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		klog.V(2).InfoS("redirecting blob request to upstream registry", "path", rPath, "redirect", redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	}
}

func redirectUpstream(rc RegistryConfig, originalPath string, ipInfo cloudcidrs.IPInfo, defaultRegistry Registry) (string, error) {
	reg := defaultRegistry

	// Determine endpoint based on provider and region
	switch ipInfo.Cloud {
	case cloudcidrs.AZ:
		klog.V(2).Infof("Redirecting to Azure endpoint")
		reg = rc.UpstreamACR
	case cloudcidrs.GCP:
		// Use radix tree for O(k) region lookup where k = prefix length
		if mappedReg, regionType := findGCPRegistry(ipInfo.Region); mappedReg != nil {
			reg = *mappedReg
			klog.V(2).Infof("Redirecting to GCP %s endpoint", regionType)
		} else {
			klog.V(2).Infof("Unknown GCP region %s, using default gcp %s", ipInfo.Region, rc.UpstreamUsGAR.Endpoint)
			reg = rc.UpstreamUsGAR
		}
	default:
		klog.V(2).Infof("Redirecting to default endpoint %v", reg)
	}

	// Build the redirect URL
	redirectUrl, err := url.JoinPath(reg.Endpoint, "/v2/", reg.Namespace, strings.TrimPrefix(originalPath, "/v2"))
	if err != nil {
		return "", err
	}
	return redirectUrl, nil
}
