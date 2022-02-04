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
	"flag"
	"net/http"
	"os"
	"strings"

	"k8s.io/klog/v2"
)

func main() {
	// klog setup
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// cloud run expects us to listen to HTTP on $PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// actually serve traffic
	klog.InfoS("listening", "port", port)
	if err := http.ListenAndServe(":"+port, http.HandlerFunc(handler)); err != nil {
		klog.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// right now we just need to serve a redirect, but all
	// valid requests should be at /v2/ or /v1/, so we leave this check
	// in the future we will selectively redirect clients to different copies
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/v2/"):
		doV2(w, r)
	case strings.HasPrefix(path, "/v1/"):
		doV1(w, r)
	default:
		klog.V(2).InfoS("unknown request", "path", path)
		http.NotFound(w, r)
	}
}

func doV2(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	klog.V(2).InfoS("v2 request", "path", path)
	http.Redirect(w, r, "https://k8s.gcr.io"+path, http.StatusPermanentRedirect)
}

// TODO: should we even be supporting v1 API anymore?
func doV1(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	klog.V(2).InfoS("v1 request", "path", path)
	http.Redirect(w, r, "https://k8s.gcr.io"+path, http.StatusPermanentRedirect)
}
