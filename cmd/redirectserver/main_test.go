//go:build !nointegration
// +build !nointegration

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
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"k8s.io/registry.k8s.io/internal/integration"
)

// TestIntegrationMain tests the entire, built binary with an integration
// test, pulling images with crane
func TestIntegrationMain(t *testing.T) {
	ctx := context.Background()

	rootDir, err := integration.ModuleRootDir()
	if err != nil {
		t.Fatalf("Failed to detect module root dir: %v", err)
	}

	// build binary
	buildCmd := exec.Command("make", "redirectserver")
	buildCmd.Dir = rootDir
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build redirectserver for integration testing: %v", err)
	}

	// start server in background
	testPort := "61337"
	testAddr := "localhost:" + testPort
	serverErrChan := make(chan error)
	cmdContext, serverCancel := context.WithCancel(context.TODO())
	serverCmd := exec.CommandContext(cmdContext, "bin/redirectserver")
	serverCmd.Dir = rootDir
	serverCmd.Env = append(serverCmd.Env, "HOME="+os.Getenv("HOME"))
	serverCmd.Env = append(serverCmd.Env, "USER="+os.Getenv("USER"))
	serverCmd.Env = append(serverCmd.Env, "PORT="+testPort)
	serverCmd.Stderr = os.Stderr
	defer serverCancel()
	go func() {
		serverErrChan <- serverCmd.Start()
		serverErrChan <- serverCmd.Wait()
	}()

	// wait for server to be up and running
	startErr := <-serverErrChan
	if startErr != nil {
		t.Fatalf("Failed to start redirectserver: %v", startErr)
	}
	if !tryUntil(time.Now().Add(time.Second), func() bool {
		_, err := http.Get("http://" + testAddr + "/")
		return err == nil
	}) {
		t.Fatal("timed out waiting for redirectserver to be ready")
	}

	// TODO: fake being on AWS
	testGet := func(ctx context.Context, relativePath string, wantStatusCode int, wantContent string) {
		url := "http://" + testAddr + "/" + relativePath
		// Do not follow redirects
		httpClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Errorf("error building HTTP requst for GET %q: %v", url, err)
			return
		}
		response, err := httpClient.Do(req)
		if err != nil {
			t.Errorf("GET for %q failed: %v", url, err)
			return
		}
		defer response.Body.Close()
		b, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("error reading response body for %q failed: %v", url, err)
			return
		}
		body := string(b)
		if response.StatusCode != wantStatusCode {
			t.Errorf("GET for %q gave unexpected response code: got %v, want %v.  Body=%q", url, response.StatusCode, wantStatusCode, body)
			return
		}
		if body != wantContent {
			t.Errorf("GET for %q gave unexpected body: got %q, want %q", url, body, wantContent)
			return
		}
	}

	// test fetching sha256, should be served directly
	testGet(ctx, "binaries/kops/1.24.3/darwin/arm64/kops.sha256", http.StatusOK, "1a946447cdd9baeaff6780ac05f3c1fbb9486c57436a31b4476ea2b161f8739a\n")

	// test fetching non-hash, should be served via redirect
	testGet(ctx, "binaries/kops/1.24.3/darwin/arm64/kops", http.StatusTemporaryRedirect, "<a href=\"https://artifacts.k8s.io/binaries/kops/1.24.3/darwin/arm64/kops\">Temporary Redirect</a>.\n\n")

	// test fetching non-existent, should be served via redirect
	testGet(ctx, "binaries/not-a-file", http.StatusTemporaryRedirect, "<a href=\"https://artifacts.k8s.io/binaries/not-a-file\">Temporary Redirect</a>.\n\n")

	// test fetching non-existent hash, should give not-found
	testGet(ctx, "binaries/not-a-file.sha256", http.StatusNotFound, "Not Found\n")

	// test fetching privacy, should be served via redirect
	testGet(ctx, "privacy", http.StatusTemporaryRedirect, "<a href=\"https://www.linuxfoundation.org/privacy-policy/\">Temporary Redirect</a>.\n\n")

	// test fetching root, should be served via redirect
	testGet(ctx, "", http.StatusTemporaryRedirect, "<a href=\"https://github.com/kubernetes/registry.k8s.io\">Temporary Redirect</a>.\n\n")

	// we're done, cleanup
	if err := serverCmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("failed to signal archeio: %v", err)
	}
	if err := <-serverErrChan; err != nil {
		t.Fatalf("archeio did not exit cleanly: %v", err)
	}
}

// helper that calls `try()` in a loop until the deadline `until`
// has passed or `try()`returns true, returns whether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
		time.Sleep(time.Millisecond * 10)
	}
	return false
}
