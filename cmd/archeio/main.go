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

package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, http.HandlerFunc(handler)); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/v2/"):
		doV2(w, r)
	case strings.HasPrefix(path, "/v1/"):
		doV1(w, r)
	default:
		log.Printf("unknown request: %q", path)
		http.NotFound(w, r)
	}
}

var reBlob = regexp.MustCompile("^/v2/.*/blobs/sha256:[0-9a-f]{64}$")

func doV2(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if reBlob.MatchString(path) {
		// Blob requests are the fun ones.
		log.Printf("v2 blob request: %q", path)
		//FIXME: look up the best backend
		http.Redirect(w, r, "https://k8s.gcr.io"+path, http.StatusTemporaryRedirect)
		return
	}

	// Anything else (manifests in particular) go to the canonical registry.
	log.Printf("v2 request: %q", path)
	http.Redirect(w, r, "https://k8s.gcr.io"+path, http.StatusPermanentRedirect)
}

func doV1(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Printf("v1 request: %q", path)
	//FIXME: look up backend?
	http.Redirect(w, r, "https://k8s.gcr.io"+path, http.StatusPermanentRedirect)
}
