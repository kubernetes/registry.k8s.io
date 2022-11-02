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
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/cmd/archeio/app"
	"k8s.io/registry.k8s.io/pkg/net/cidrs"
	"k8s.io/registry.k8s.io/pkg/net/cidrs/aws"
)

type MirrorConfig struct {
	CanonicalLocation GCSKey
	// CanonicalFallback is the fallback URL direct to a public bucket or similar,
	// this is used when we're having problems finding a redirect bucket.
	CanonicalFallback string

	InfoURL    string
	PrivacyURL string
}

type Server struct {
	MirrorConfig

	// contentCache stores the content of files, in particular the hash files
	contentCache *contentCache

	// regionMapper maps from a client IP to a cloud & region
	regionMapper cidrs.IPMapper[string]

	// mirrorCache records whether the mirrors have the various files.
	// We use a lookup so we don't require 100% perfect replication to all mirrors.
	mirrorCache app.BlobChecker
}

func NewServer(cfg MirrorConfig, storageClient *storage.Client, mirrorCache app.BlobChecker) *Server {
	s := &Server{
		MirrorConfig: cfg,
	}
	// cache of hash files
	s.contentCache = newContentCache(storageClient)
	// initialize map of clientIP to AWS region
	s.regionMapper = aws.NewAWSRegionMapper()
	// cache of whether the mirrors have blobs
	s.mirrorCache = mirrorCache

	return s
}

// ServeHTTP is the main entry point
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only allow GET, HEAD; we don't allow
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Only GET and HEAD are allowed.", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/binaries/"):
		s.serveBinaries(w, r)
	case path == "/":
		http.Redirect(w, r, s.InfoURL, http.StatusTemporaryRedirect)
	case strings.HasPrefix(path, "/privacy"):
		http.Redirect(w, r, s.PrivacyURL, http.StatusTemporaryRedirect)
	default:
		klog.V(2).InfoS("unknown request", "path", path)
		http.NotFound(w, r)
	}
}

func (s *Server) serveBinaries(w http.ResponseWriter, r *http.Request) {
	rPath := r.URL.Path

	serveDirectly := false

	suffix := getSuffix(rPath)
	switch strings.ToLower(suffix) {
	case ".sha1", ".sha224", ".sha256", ".sha384", ".sha512", ".md5":
		// Hashes are not redirected, both for security and because they're so small
		serveDirectly = true
	case ".gpg":
		// Like hashes, we serve small security-critical key/signature information directly
		serveDirectly = true
	}

	if serveDirectly {
		s.serveDirectly(w, r)
		return
	}

	// for blob requests, check the client IP and determine the best backend
	clientIP, err := app.GetClientIP(r)
	if err != nil {
		// this should not happen
		klog.ErrorS(err, "failed to get client IP")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// check if client is known to be coming from an AWS region
	awsRegion, ipIsKnown := s.regionMapper.GetIP(clientIP)
	if !ipIsKnown {
		// no region match, redirect to fallback location
		klog.V(2).InfoS("region not known; redirecting request to fallback location", "path", rPath)
		s.redirectToCanonical(w, r)
		return
	}

	// check if blob is available in our S3 bucket for the region
	mirrorBase := awsRegionToS3URL(awsRegion)
	mirrorURL := urlJoin(mirrorBase, rPath)
	if s.mirrorCache.BlobExists(mirrorURL, mirrorBase, rPath) {
		// blob known to be available in S3, redirect client there
		klog.V(2).InfoS("redirecting blob request to mirror", "path", rPath, "mirror", mirrorBase)
		http.Redirect(w, r, mirrorURL, http.StatusTemporaryRedirect)
		return
	}

	// fall back to redirect to upstream
	klog.V(2).InfoS("blob not found; redirecting blob request to fallback location", "path", rPath)
	s.redirectToCanonical(w, r)
}

func (s *Server) serveDirectly(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rPath := r.URL.Path

	u := s.CanonicalLocation.Join(rPath)
	data, err := s.contentCache.GetContents(ctx, u)
	if err != nil {
		if os.IsNotExist(err) {
			klog.V(2).InfoS("object not found in canonical location", "url", u)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		klog.V(2).ErrorS(err, "error reading object in canonical location", "url", u)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: Headers
	// 	< HTTP/1.1 200 OK
	// < X-GUploader-UploadID: ADPycdvBD5IWSlLZPTD17ArlCH9Aq8rNp-XRiefxNuI3e1rq2ZSZEs_E8nZ8Zg5wJcn47E9udEoCiGff6zCrS2WpkjNS
	// < x-goog-generation: 1666089540728742
	// < x-goog-metageneration: 1
	// < x-goog-stored-content-encoding: identity
	// < x-goog-stored-content-length: 65
	// < x-goog-hash: crc32c=aET7Hg==
	// < x-goog-hash: md5=moj7swqz3Hi/1S99ecNCsA==
	// < x-goog-storage-class: STANDARD
	// < Accept-Ranges: bytes
	// < Content-Length: 65
	// < Server: UploadServer
	// < Date: Tue, 01 Nov 2022 20:28:07 GMT
	// < Cache-Control: public,max-age=86400
	// < Last-Modified: Tue, 18 Oct 2022 10:39:00 GMT
	// < ETag: "9a88fbb30ab3dc78bfd52f7d79c342b0"
	// < Content-Type: text/plain; charset=utf-8
	// < Age: 24263

	klog.V(2).InfoS("serving object from canonical location", "url", u)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	ignoreErrorTooLate(err)
}

func ignoreErrorTooLate(_ error) {
	// It's too late to do anything about errors, we're written the response etc
}

func (s *Server) redirectToCanonical(w http.ResponseWriter, r *http.Request) {
	rPath := r.URL.Path

	// Our canonical bucket isn't currently publicly readable.
	// We could construct a signedurl here, but this is a bit of a pain because we need a keypair.
	// Maybe: https://gist.github.com/pdecat/80f21e36583420abbfdeae0494a53501
	// u := s.CanonicalLocation.Join(rPath)
	// opts := storage.SignedURLOptions{}
	// s.storageClient.Bucket(u.Bucket).SignedURL(u.ObjectKey, opts)

	// Instead, we redirect to a "primary bucket" that is assumed public, for now.

	redirectURL := urlJoin(s.CanonicalFallback, rPath)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// urlJoin performs simple url joining without canonicalization
func urlJoin(base string, path string) string {
	var s strings.Builder
	s.WriteString(base)
	if !strings.HasSuffix(base, "/") {
		s.WriteString("/")
	}
	s.WriteString(strings.TrimPrefix(path, "/"))
	return s.String()
}

// getSuffix returns the file suffix (the component from the last dot character)
func getSuffix(urlPath string) string {
	lastDot := strings.LastIndex(urlPath, ".")
	if lastDot == -1 {
		return ""
	}
	return urlPath[lastDot:]
}
