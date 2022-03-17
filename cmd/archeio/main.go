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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

type metrics struct {
	Requests       *prometheus.CounterVec
	ActiveRequests *prometheus.GaugeVec
}

func newMetrics() *metrics {
	m := &metrics{
		Requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "requests",
				Help: "The processed requests",
			},
			[]string{"code", "method", "version"},
		),
		ActiveRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_requests",
				Help: "The amount of requests that are current",
			},
			[]string{},
		),
	}
	prometheus.MustRegister(m.Requests)
	prometheus.MustRegister(m.ActiveRequests)
	return m
}

var (
	serverMetrics = newMetrics()
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
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "2112"
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// actually serve traffic
	klog.InfoS("listening", "port", port)
	handlerFromFunc := http.HandlerFunc(handler)
	handlerWithMetrics := promhttp.InstrumentHandlerInFlight(
		serverMetrics.ActiveRequests.With(prometheus.Labels{}),
		handlerFromFunc,
	)
	server := &http.Server{
		Addr:        ":" + port,
		Handler:     handlerWithMetrics,
		ReadTimeout: 10 * time.Second,
	}

	// start serving
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()
	klog.Infof("Server started")

	klog.InfoS("metrics", "port", metricsPort)
	metricsServer := &http.Server{
		Addr:        ":" + metricsPort,
		Handler:     promhttp.Handler(),
		ReadTimeout: 10 * time.Second,
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()
	klog.Infof("Metrics server started")

	// Graceful shutdown
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	klog.Infof("Shutting down server")
	if err := server.Shutdown(ctx); err != nil {
		klog.Fatalf("Server didn't exit gracefully %v", err)
	}
	if err := metricsServer.Shutdown(ctx); err != nil {
		klog.Fatalf("Metrics server didn't exit gracefully %v", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// right now we just need to serve a redirect, but all
	// valid requests should be at /v2/ or /v1/, so we leave this check
	// in the future we will selectively redirect clients to different copies
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/v2/"):
		serverMetrics.Requests.With(prometheus.Labels{"code": fmt.Sprintf("%v", http.StatusPermanentRedirect), "method": "GET", "version": "v2"}).Inc()
		doV2(w, r)
	case strings.HasPrefix(path, "/v1/"):
		serverMetrics.Requests.With(prometheus.Labels{"code": fmt.Sprintf("%v", http.StatusPermanentRedirect), "method": "GET", "version": "v2"}).Inc()
		doV1(w, r)
	default:
		klog.V(2).InfoS("unknown request", "path", path)
		serverMetrics.Requests.With(prometheus.Labels{"code": fmt.Sprintf("%v", http.StatusNotFound), "method": "GET", "version": "unknown"}).Inc()
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
