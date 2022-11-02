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
	// setup crane
	rootDir, err := integration.ModuleRootDir()
	if err != nil {
		t.Fatalf("Failed to detect module root dir: %v", err)
	}
	// NOTE: also ensures rootDir/bin is in front of $PATH
	if err := integration.EnsureCrane(rootDir); err != nil {
		t.Fatalf("Failed to ensure crane: %v", err)
	}

	// build binary
	buildCmd := exec.Command("make", "archeio")
	buildCmd.Dir = rootDir
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build archeio for integration testing: %v", err)
	}

	// start server in background
	testPort := "61337"
	testAddr := "localhost:" + testPort
	serverErrChan := make(chan error)
	cmdContext, serverCancel := context.WithCancel(context.TODO())
	serverCmd := exec.CommandContext(cmdContext, "archeio")
	serverCmd.Env = append(serverCmd.Env, "PORT="+testPort)
	// serverCmd.Stderr = os.Stderr
	defer serverCancel()
	go func() {
		serverErrChan <- serverCmd.Start()
		serverErrChan <- serverCmd.Wait()
	}()

	// wait for server to be up and running
	startErr := <-serverErrChan
	if startErr != nil {
		t.Fatalf("Failed to start archeio: %v", startErr)
	}
	if !tryUntil(time.Now().Add(time.Second), func() bool {
		_, err := http.Get("http://" + testAddr + "/v2/")
		return err == nil
	}) {
		t.Fatal("timed out waiting for archeio to be ready")
	}

	// TODO: fake being on AWS
	testPull := func(image string) {
		// nolint:gosec // this is not user suplied input ...
		cmd := exec.Command("crane", "pull", testAddr+"/"+image, os.DevNull)
		//cmd.Stderr = os.Stderr
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("pull for %q failed: %v", image, err)
			t.Error("output: ")
			t.Error(string(out))
			t.Fail()
		}
	}

	// test pulling pause image
	// TODO: test pulling more things
	testPull("pause:3.1")

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
