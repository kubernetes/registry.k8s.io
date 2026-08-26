//go:build linux && !noe2e

/*
Copyright 2023 The Kubernetes Authors.

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

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestE2EContainerdPull(t *testing.T) {
	t.Parallel()
	containerdVersions := []string{"1.7.34", "2.0.11", "2.3.3"}
	for i := range containerdVersions {
		containerdVersion := containerdVersions[i]
		t.Run("v"+containerdVersion, func(t *testing.T) {
			testE2EContainerdPull(t, containerdVersion)
		})
	}
}

func testE2EContainerdPull(t *testing.T, containerdVersion string) {
	t.Parallel()
	// install containerd and image puller tool
	installDir := filepath.Join(binDir, "containerd-"+containerdVersion)
	// nolint:gosec
	installCmd := exec.Command(filepath.Join(repoRoot, "hack", "tools", "e2e-setup-containerd.sh"))
	installCmd.Env = append(installCmd.Env,
		"CONTAINERD_VERSION="+containerdVersion,
		"CONTAINERD_INSTALL_DIR="+installDir,
		"CONTAINERD_ARCH="+runtime.GOARCH,
	)
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		t.Fatalf("Failed to install containerd: %v", err)
	}

	// start rootless containerd, which only needs to be able to pull images
	tmpDir, err := os.MkdirTemp("", "containerd")
	if err != nil {
		t.Fatalf("Failed to setup tmpdir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	socketAddress := filepath.Join(tmpDir, "containerd.sock")

	// Generate config at runtime so each test instance has isolated paths
	configPath := filepath.Join(tmpDir, "containerd-config.toml")
	nriSocketPath := filepath.Join(tmpDir, "nri.sock")
	configContent := fmt.Sprintf(`# Generated at test runtime for isolated paths
[grpc]
  uid = %d
  gid = %d

# Set NRI socket path to tmpDir so parallel tests don't conflict
[plugins."io.containerd.nri.v1.nri"]
  socket_path = %q
`, os.Getuid(), os.Getgid(), nriSocketPath)
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write containerd config: %v", err)
	}

	// nolint:gosec
	containerdCmd := exec.Command(
		filepath.Join(installDir, "containerd"),
		"--config="+configPath,
		"--root="+filepath.Join(tmpDir, "root"),
		"--state="+filepath.Join(tmpDir, "state"),
		"--address="+socketAddress,
		"--log-level=trace",
	)
	// Capture containerd logs for debugging on failure
	var containerdLogs bytes.Buffer
	containerdCmd.Stderr = &containerdLogs
	if err := containerdCmd.Start(); err != nil {
		t.Fatalf("Failed to start containerd: %v", err)
	}
	// Channel to detect early containerd exit
	containerdExited := make(chan error, 1)
	go func() {
		containerdExited <- containerdCmd.Wait()
	}()
	t.Cleanup(func() {
		// Check if already exited
		select {
		case <-containerdExited:
			// Already exited, nothing to do
			return
		default:
		}
		if err := containerdCmd.Process.Signal(os.Interrupt); err != nil {
			t.Logf("failed to signal containerd: %v", err)
			return
		}
		// kill if it doesn't exit gracefully after 1s
		select {
		case <-containerdExited:
			// exited
		case <-time.After(time.Second):
			// timed out
			if err := containerdCmd.Process.Kill(); err != nil {
				t.Logf("Failed to kill containerd: %v", err)
			}
			<-containerdExited // Wait for goroutine to complete
		}
	})

	// wait for containerd to be ready (max ~55 seconds: 0+1+4+9+16+25)
	containerdReady := false
	for i := 0; i < 6; i++ {
		// Check if containerd exited early
		select {
		case err := <-containerdExited:
			t.Fatalf("containerd exited unexpectedly: %v\nLogs:\n%s", err, containerdLogs.String())
		default:
		}
		// nolint:gosec
		if err := exec.Command(filepath.Join(installDir, "ctr"), "--address="+socketAddress, "version").Run(); err == nil {
			containerdReady = true
			break
		}
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if !containerdReady {
		// Check one more time if it exited
		select {
		case err := <-containerdExited:
			t.Fatalf("containerd exited while waiting for ready: %v\nLogs:\n%s", err, containerdLogs.String())
		default:
		}
		t.Fatalf("Failed to wait for containerd to be ready after ~55s\nLogs:\n%s", containerdLogs.String())
	}

	// pull test images
	for i := range testCases {
		tc := &testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			// nolint:gosec
			pullCmd := exec.Command(filepath.Join(installDir, "ctr"), "--address="+socketAddress, "content", "fetch", tc.Ref())
			testPull(t, tc, pullCmd)
		})
	}
}
