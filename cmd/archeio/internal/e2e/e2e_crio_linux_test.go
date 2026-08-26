//go:build linux && !noe2e

/*
Copyright 2026 The Kubernetes Authors.

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

func TestE2ECrioPull(t *testing.T) {
	t.Parallel()
	crioVersions := []string{"1.34.12", "1.35.7", "1.36.4"}
	for i := range crioVersions {
		crioVersion := crioVersions[i]
		t.Run("v"+crioVersion, func(t *testing.T) {
			testE2ECrioPull(t, crioVersion)
		})
	}
}

func testE2ECrioPull(t *testing.T, crioVersion string) {
	t.Parallel()
	// install cri-o static bundle, which ships crio, crictl, conmon and crun
	installDir := filepath.Join(binDir, "crio-"+crioVersion)
	// nolint:gosec
	installCmd := exec.Command(filepath.Join(repoRoot, "hack", "tools", "e2e-setup-crio.sh"))
	installCmd.Env = append(installCmd.Env,
		"CRIO_VERSION="+crioVersion,
		"CRIO_INSTALL_DIR="+installDir,
		"CRIO_ARCH="+runtime.GOARCH,
	)
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		t.Fatalf("Failed to install cri-o: %v", err)
	}

	// start crio, which only needs to be able to pull images
	tmpDir, err := os.MkdirTemp("", "crio")
	if err != nil {
		t.Fatalf("Failed to setup tmpdir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	socketAddress := filepath.Join(tmpDir, "crio.sock")
	runtimeEndpoint := "unix://" + socketAddress

	// signature verification is not under test here
	policyPath := filepath.Join(tmpDir, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"default":[{"type":"insecureAcceptAnything"}]}`), 0600); err != nil {
		t.Fatalf("Failed to write signature policy: %v", err)
	}

	// the local test registry is plain HTTP and CRI-O >= 1.35 no longer
	// implicitly trusts localhost, so mark the endpoint insecure
	registriesPath := filepath.Join(tmpDir, "registries.conf")
	registriesContent := fmt.Sprintf("[[registry]]\nlocation = %q\ninsecure = true\n", endpoint)
	if err := os.WriteFile(registriesPath, []byte(registriesContent), 0600); err != nil {
		t.Fatalf("Failed to write registries config: %v", err)
	}

	// Generate config at runtime so each test instance has isolated paths
	// vfs avoids requiring overlayfs privileges in the test environment
	configPath := filepath.Join(tmpDir, "crio.conf")
	configContent := fmt.Sprintf(`# Generated at test runtime for isolated paths
[crio]
root = %q
runroot = %q
log_dir = %q
version_file = %q
version_file_persist = ""
clean_shutdown_file = %q
storage_driver = "vfs"

[crio.api]
listen = %q

[crio.runtime]
conmon = %q
pinns_path = %q
default_runtime = "crun"
namespaces_dir = %q
container_exits_dir = %q
container_attach_socket_dir = %q
# unprivileged CI cannot use iptables/nftables and we only pull images
disable_hostport_mapping = true

[crio.runtime.runtimes.crun]
runtime_path = %q
runtime_root = %q

[crio.image]
signature_policy = %q
`,
		filepath.Join(tmpDir, "root"),
		filepath.Join(tmpDir, "runroot"),
		filepath.Join(tmpDir, "log"),
		filepath.Join(tmpDir, "version"),
		filepath.Join(tmpDir, "clean.shutdown"),
		socketAddress,
		filepath.Join(installDir, "bin", "conmon"),
		filepath.Join(installDir, "bin", "pinns"),
		filepath.Join(tmpDir, "ns"),
		filepath.Join(tmpDir, "exits"),
		filepath.Join(tmpDir, "attach"),
		filepath.Join(installDir, "bin", "crun"),
		filepath.Join(tmpDir, "crun"),
		policyPath,
	)
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write crio config: %v", err)
	}

	// nolint:gosec
	crioCmd := exec.Command(
		filepath.Join(installDir, "bin", "crio"),
		"--config="+configPath,
		// crio sets the path in its SystemContext, so the
		// CONTAINERS_REGISTRIES_CONF env var would be ignored
		"--registries-conf="+registriesPath,
		"--log-level=debug",
	)
	crioCmd.Env = append(os.Environ(), "XDG_RUNTIME_DIR="+tmpDir)
	// Capture crio logs for debugging on failure
	var crioLogs bytes.Buffer
	crioCmd.Stderr = &crioLogs
	if err := crioCmd.Start(); err != nil {
		t.Fatalf("Failed to start crio: %v", err)
	}
	// Channel to detect early crio exit
	crioExited := make(chan error, 1)
	go func() {
		crioExited <- crioCmd.Wait()
	}()
	t.Cleanup(func() {
		// Check if already exited
		select {
		case <-crioExited:
			// Already exited, nothing to do
			return
		default:
		}
		if err := crioCmd.Process.Signal(os.Interrupt); err != nil {
			t.Logf("failed to signal crio: %v", err)
			return
		}
		// kill if it doesn't exit gracefully after 1s
		select {
		case <-crioExited:
			// exited
		case <-time.After(time.Second):
			// timed out
			if err := crioCmd.Process.Kill(); err != nil {
				t.Logf("Failed to kill crio: %v", err)
			}
			<-crioExited // Wait for goroutine to complete
		}
	})

	crictl := filepath.Join(installDir, "bin", "crictl")
	// wait for crio to be ready (max ~55 seconds: 0+1+4+9+16+25)
	crioReady := false
	for i := 0; i < 6; i++ {
		// Check if crio exited early
		select {
		case err := <-crioExited:
			t.Fatalf("crio exited unexpectedly: %v\nLogs:\n%s", err, crioLogs.String())
		default:
		}
		// nolint:gosec
		if err := exec.Command(crictl, "--runtime-endpoint="+runtimeEndpoint, "version").Run(); err == nil {
			crioReady = true
			break
		}
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if !crioReady {
		// Check one more time if it exited
		select {
		case err := <-crioExited:
			t.Fatalf("crio exited while waiting for ready: %v\nLogs:\n%s", err, crioLogs.String())
		default:
		}
		t.Fatalf("Failed to wait for crio to be ready after ~55s\nLogs:\n%s", crioLogs.String())
	}

	// pull test images
	for i := range testCases {
		tc := &testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			// CRI-O reports the platform manifest digest rather than the index
			// digest, so verify by pulling by digest instead of matching output
			pullTC := testCase{Name: tc.Name}
			// nolint:gosec
			pullCmd := exec.Command(crictl,
				"--runtime-endpoint="+runtimeEndpoint,
				"--image-endpoint="+runtimeEndpoint,
				"pull", tc.Ref()+"@"+tc.Digest,
			)
			testPull(t, &pullTC, pullCmd)
		})
	}
}
