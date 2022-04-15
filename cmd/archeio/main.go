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
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// make it possible to override k8s.gcr.io without rebuilding in the future
	upstreamRegistry := os.Getenv("UPSTREAM_REGISTRY")
	if upstreamRegistry == "" {
		upstreamRegistry = "https://k8s.gcr.io"
	}

	// actually serve traffic
	klog.InfoS("listening", "port", port)
	server := &http.Server{
		Addr:        ":" + port,
		Handler:     MakeHandler(upstreamRegistry),
		ReadTimeout: 10 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// start serving
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()
	klog.Infof("Server started")

	// Graceful shutdown
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		klog.Fatalf("Server didn't exit gracefully %v", err)
	}
}
