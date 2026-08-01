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
	"strconv"
	"syscall"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/cmd/archeio/internal/app"
)

func main() {
	// klog setup
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// route klog through zap for JSON structured logs
	// Cloud Logging parses JSON on stderr into jsonPayload with these keys
	// https://cloud.google.com/logging/docs/structured-logging
	setupJSONLogging()

	// cloud run expects us to listen to HTTP on $PORT
	// https://cloud.google.com/run/docs/container-contract#port
	port := getEnv("PORT", "8080")

	registryConfig := app.RegistryConfig{
		UpstreamRegistryEndpoint:  getEnv("UPSTREAM_REGISTRY_ENDPOINT", "https://us-central1-docker.pkg.dev"),
		UpstreamRegistryPath:      getEnv("UPSTREAM_REGISTRY_PATH", "k8s-artifacts-prod/images"),
		SignatureUpstreamEndpoint: getEnv("SIGNATURE_UPSTREAM_ENDPOINT", ""),
		InfoURL:                   "https://github.com/kubernetes/registry.k8s.io",
		PrivacyURL:                "https://www.linuxfoundation.org/privacy-policy/",
		DefaultAWSBaseURL:         getEnv("DEFAULT_AWS_BASE_URL", "https://d1be1w964nk82h.cloudfront.net"),
	}

	// configure server with reasonable timeout
	// we only serve redirects, 10s should be sufficient
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.MakeHandler(registryConfig),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// signal handler for graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// start serving
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()
	klog.InfoS("listening", "port", port)
	klog.InfoS("registry", "configuration", registryConfig)

	// Graceful shutdown
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		klog.Fatalf("Server didn't exit gracefully %v", err)
	}
}

// getEnv returns defaultValue if key is not set, else the value of os.LookupEnv(key)
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// setupJSONLogging replaces klog's default text output with zap JSON output
// via zapr, using Cloud Logging's special field names so entries are parsed
// into severity / message / timestamp instead of a text blob
//
// the klog -v flag still controls verbosity: klog.V(n) maps to zap level -n
func setupJSONLogging() {
	verbosity := 0
	if f := flag.Lookup("v"); f != nil {
		if v, err := strconv.Atoi(f.Value.String()); err == nil {
			verbosity = v
		}
	}
	zc := zap.NewProductionConfig()
	zc.EncoderConfig.MessageKey = "message"
	zc.EncoderConfig.LevelKey = "severity"
	// map zap levels to Cloud Logging severities, klog.V(n) logs at zap
	// level -n which would otherwise render as unparseable "LEVEL(-n)"
	zc.EncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		if l < zapcore.InfoLevel {
			enc.AppendString("DEBUG")
			return
		}
		zapcore.CapitalLevelEncoder(l, enc)
	}
	zc.EncoderConfig.TimeKey = "timestamp"
	zc.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	// allow klog.V(verbosity) and below
	zc.Level = zap.NewAtomicLevelAt(zapcore.Level(-verbosity)) //nolint:gosec // verbosity is a small flag value
	zc.Sampling = nil                                          // never drop logs
	zLogger, err := zc.Build()
	if err != nil {
		klog.Fatalf("failed to build zap logger: %v", err)
	}
	klog.SetLogger(zapr.NewLogger(zLogger))
}
