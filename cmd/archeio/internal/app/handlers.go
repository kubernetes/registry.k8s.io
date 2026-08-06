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
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"path"
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/pkg/net/clientip"
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

type RegistryConfig struct {
	UpstreamRegistryEndpoint  string
	UpstreamRegistryPath      string
	SignatureUpstreamEndpoint string
	InfoURL                   string
	PrivacyURL                string
	DefaultAWSBaseURL         string
}

const (
	// traceIDHeader is the response header echoing the trace ID to the client
	traceIDHeader = "X-Request-ID"
	// traceIDQueryParam is appended to redirect Location URLs so access logs
	// at the redirect target (S3 / CloudFront / upstream registry) can be
	// joined back to archeio's logs. Backends must be configured to exclude
	// this parameter from cache keys.
	traceIDQueryParam = "rid"
)

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
		case path == "/token":
			serveToken(w, r)
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
	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pull
	// Blobs are at `/v2/<name>/blobs/<digest>`
	// Note that ':' cannot be contained in <name> but *must* be contained in <digest>
	// <digest> also cannot contain `/` so we can use a relatively simple and cheap regex
	// to match blob requests and capture the digest
	reBlob := regexp.MustCompile("^/v2/.*/blobs/([^/]+:[a-zA-Z0-9=_-]+)$")
	// matches cosign signature and attestation tag requests
	reCosignTag := regexp.MustCompile(`^/v2/.*/manifests/sha256-[a-f0-9]{64}\.(sig|att)$`)
	// initialize map of clientIP to AWS region
	regionMapper := cloudcidrs.NewIPMapper()
	// capture these in a http handler lambda
	return func(w http.ResponseWriter, r *http.Request) {
		rPath := r.URL.Path

		// correlation ID for tracing this request across redirects
		// a Bearer token issued by /token identifies the whole pull session,
		// falling back to a per-request ID when no token is presented
		// echoed back to the client and appended to redirect targets so
		// backend / CDN access logs can be joined with archeio's own logs
		traceID := bearerPullID(r)
		if traceID == "" {
			traceID = traceIDForRequest(r)
		}
		if traceID != "" {
			w.Header().Set(traceIDHeader, traceID)
		}

		// we only care about publicly readable GCR as the backing registry
		// or publicly readable blob storage, no real authentication is needed
		//
		// however, we use the standard registry token flow purely for trace
		// correlation: challenging the client on /v2/ makes it fetch a
		// "token" from /token (a trace ID), which the client then presents
		// on every subsequent request of the pull, giving us one consistent
		// correlation ID for the entire image pull session
		//
		// it turns out publicly readable GCR repos do not actually care about
		// the presence of a token for any API calls, and standard clients
		// strip the Authorization header when following redirects to other
		// hosts, so the correlation token never leaks to backends
		if rPath == "/v2/" || rPath == "/v2" {
			// NOTE: OCI does not require this, but the docker v2 spec include it, and GCR sets this
			// Docker distribution v2 clients may fallback to an older version if this is not set.
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			if bearerPullID(r) != "" {
				klog.V(2).InfoS("serving 200 OK for /v2/ check", "path", rPath, "traceID", traceID)
				w.WriteHeader(http.StatusOK)
				return
			}
			// challenge the client to fetch a pull correlation token
			klog.V(2).InfoS("serving 401 challenge for /v2/ check", "path", rPath, "traceID", traceID)
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+requestScheme(r)+`://`+r.Host+`/token",service="`+r.Host+`"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		// we don't support the non-standard _catalog API
		// https://github.com/kubernetes/registry.k8s.io/issues/162
		if rPath == "/v2/_catalog" {
			http.Error(w, "_catalog is not supported", http.StatusNotFound)
			return
		}

		// resolve the client IP and cloud/region info once, used for backend
		// selection on blob requests and for analytics events
		// failure to resolve is only fatal for blob requests, see below
		var ipInfo *cloudcidrs.IPInfo
		clientIP, clientIPErr := clientip.Get(r)
		if clientIPErr == nil {
			if info, ipIsKnown := regionMapper.GetIP(clientIP); ipIsKnown {
				ipInfo = &info
			}
		}

		// check if blob request
		matches := reBlob.FindStringSubmatch(rPath)
		if len(matches) != 2 {
			// check if this is a cosign signature/attestation request
			if rc.SignatureUpstreamEndpoint != "" && reCosignTag.MatchString(rPath) {
				redirectURL := signatureRedirectURL(rc, rPath)
				klog.V(2).InfoS("redirecting cosign signature request to canonical upstream", "path", rPath, "redirect", redirectURL, "traceID", traceID)
				trackPullEvent(r, traceID, "signature-upstream", ipInfo)
				http.Redirect(w, r, withTraceID(redirectURL, traceID), http.StatusTemporaryRedirect)
				return
			}
			// not a blob request so forward it to the main upstream registry
			redirectURL := upstreamRedirectURL(rc, rPath)
			klog.V(2).InfoS("redirecting manifest request to upstream registry", "path", rPath, "redirect", redirectURL, "traceID", traceID)
			trackPullEvent(r, traceID, "upstream", ipInfo)
			http.Redirect(w, r, withTraceID(redirectURL, traceID), http.StatusTemporaryRedirect)
			return
		}
		// it is a blob request, grab the hash for later
		digest := matches[1]

		// for blob requests we must know the client IP to determine the best backend
		if clientIPErr != nil {
			// this should not happen
			klog.ErrorS(clientIPErr, "failed to get client IP")
			http.Error(w, clientIPErr.Error(), http.StatusBadRequest)
			return
		}

		// if client is coming from GCP, stay in GCP
		if ipInfo != nil && ipInfo.Cloud == cloudcidrs.GCP {
			redirectURL := upstreamRedirectURL(rc, rPath)
			klog.V(2).InfoS("redirecting GCP blob request to upstream registry", "path", rPath, "redirect", redirectURL, "traceID", traceID)
			trackPullEvent(r, traceID, "upstream", ipInfo)
			http.Redirect(w, r, withTraceID(redirectURL, traceID), http.StatusTemporaryRedirect)
			return
		}

		// check if blob is available in our AWS layer storage for the region
		region := ""
		if ipInfo != nil {
			region = ipInfo.Region
		}
		bucketURL := awsRegionToHostURL(region, rc.DefaultAWSBaseURL)
		// this matches GCR's GCS layout, which we will use for other buckets
		blobURL := bucketURL + "/containers/images/" + digest
		if blobs.BlobExists(blobURL, traceID) {
			// blob known to be available in AWS, redirect client there
			// NOTE: the trace ID is appended only after the existence check
			// so cached existence keys remain stable
			klog.V(2).InfoS("redirecting blob request to AWS", "path", rPath, "digest", digest, "traceID", traceID)
			trackPullEvent(r, traceID, "s3", ipInfo)
			http.Redirect(w, r, withTraceID(blobURL, traceID), http.StatusTemporaryRedirect)
			return
		}

		// fall back to redirect to upstream
		redirectURL := upstreamRedirectURL(rc, rPath)
		klog.V(2).InfoS("redirecting blob request to upstream registry", "path", rPath, "redirect", redirectURL, "traceID", traceID)
		trackPullEvent(r, traceID, "upstream", ipInfo)
		http.Redirect(w, r, withTraceID(redirectURL, traceID), http.StatusTemporaryRedirect)
	}
}

func upstreamRedirectURL(rc RegistryConfig, originalPath string) string {
	return rc.UpstreamRegistryEndpoint + path.Join("/v2/", rc.UpstreamRegistryPath, strings.TrimPrefix(originalPath, "/v2"))
}

func signatureRedirectURL(rc RegistryConfig, originalPath string) string {
	return rc.SignatureUpstreamEndpoint + path.Join("/v2/", rc.UpstreamRegistryPath, strings.TrimPrefix(originalPath, "/v2"))
}

// randRead is rand.Read, injectable for testing the failure path
var randRead = rand.Read

// GenerateTraceID creates a compliant 128-bit trace ID
func GenerateTraceID() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := randRead(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// reTraceID matches a W3C trace-context compliant 128-bit hex trace ID
var reTraceID = regexp.MustCompile("^[0-9a-fA-F]{32}$")

// traceIDForRequest returns the trace ID used to correlate this request's
// logs with the follow-up request the client makes to the redirect target.
//
// It prefers trace IDs already assigned by tracing infrastructure in front
// of archeio: W3C traceparent, then X-Cloud-Trace-Context (set by Cloud Run's
// HTTPS frontend even when the client sends nothing), falling back to
// generating a fresh ID.
//
// The returned value is either empty or guaranteed to be 32 hex characters,
// safe to embed in URLs and headers.
func traceIDForRequest(r *http.Request) string {
	// W3C trace-context: traceparent: <version>-<trace-id>-<parent-id>-<flags>
	if tp := r.Header.Get("traceparent"); tp != "" {
		parts := strings.SplitN(tp, "-", 4)
		if len(parts) == 4 && reTraceID.MatchString(parts[1]) {
			return parts[1]
		}
	}
	// X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=OPTIONS
	if xctc := r.Header.Get("X-Cloud-Trace-Context"); xctc != "" {
		id, _, _ := strings.Cut(xctc, "/")
		if reTraceID.MatchString(id) {
			return id
		}
	}
	id, err := GenerateTraceID()
	if err != nil {
		// trace correlation is best-effort, don't fail serving
		klog.ErrorS(err, "failed to generate trace ID")
		return ""
	}
	return id
}

// withTraceID appends traceID to redirectURL as a query parameter
// traceID must be pre-validated (see traceIDForRequest), empty is a no-op
func withTraceID(redirectURL, traceID string) string {
	if traceID == "" {
		return redirectURL
	}
	sep := "?"
	if strings.Contains(redirectURL, "?") {
		sep = "&"
	}
	return redirectURL + sep + traceIDQueryParam + "=" + traceID
}

// serveToken implements a minimal docker/distribution token endpoint
// https://distribution.github.io/distribution/spec/auth/token/
//
// The "token" is NOT authentication: it is a pull-session correlation ID.
// Clients present it as `Authorization: Bearer <id>` on every subsequent
// request of the pull, which lets us correlate all requests in one image
// pull without any client changes.
func serveToken(w http.ResponseWriter, r *http.Request) {
	// reuse infrastructure-assigned trace ID for the session when available
	pullID := traceIDForRequest(r)
	if pullID == "" {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// scope ties the pull ID to the repository being pulled in our logs
	klog.V(2).InfoS("issuing pull correlation token",
		"traceID", pullID,
		"scope", r.URL.Query().Get("scope"),
		"service", r.URL.Query().Get("service"),
	)
	w.Header().Set(traceIDHeader, pullID)
	w.Header().Set("Content-Type", "application/json")
	// pullID is validated 32-hex (see traceIDForRequest), safe to embed
	_, _ = w.Write([]byte(`{"token":"` + pullID + `","expires_in":3600}`))
}

// bearerPullID extracts and validates a pull correlation ID from the
// Authorization header, returning "" if absent or invalid
//
// NOTE: the value is client-controlled, it MUST be strictly validated
func bearerPullID(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !reTraceID.MatchString(token) {
		return ""
	}
	return token
}

// requestScheme returns the external scheme for the request, respecting
// the X-Forwarded-Proto header set by Cloud Run / load balancers
func requestScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "http" || proto == "https" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
