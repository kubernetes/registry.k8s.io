//go:build !noe2e
// +build !noe2e

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

// e2e contains end-to-end tests for registry.k8s.io
package e2e

/*
	This exists to test against the staging instance of the registry.

	Compare to cmd/archeio/main_test.go which exists to integration test
	a local instance. There is much overlap but they serve different purposes
	and cover different aspects.

	The integration tests can run quickly in presubmit and leverage faking
	locations to cover more codepaths. They do not however cover all interactions
	with actually deployed infrastructure including e.g. the loadbalancer and
	WAF rules in front of the deployed instances.

	These tests instead will run from multiple locations and cover the actual
	production-like infrastructure but cannot fake IP addr there by design as
	we accurately determine IP there and wouldn't want clients (ab)using this.

	These tests are still expected to be quick and cheap and only cover clients
	we can run in a containerized, non-privileged environment.

	We have other coverage, see cmd/archeio/docs/testing.md
*/

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/registry.k8s.io/internal/integration"
)

var endpoint = "registry-sandbox.k8s.io"

type testCase struct {
	Name   string
	Digest string
}

func (tc *testCase) Ref() string {
	return endpoint + "/" + tc.Name
}

var testCases = []testCase{
	{Name: "pause:3.1", Digest: "sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea"},
	{Name: "pause:3.2", Digest: "sha256:927d98197ec1141a368550822d18fa1c60bdae27b78b0c004f705f548c07814f"},
	{Name: "pause:3.5", Digest: "sha256:1ff6c18fbef2045af6b9c16bf034cc421a29027b800e4f9b68ae9b1cb3e9ae07"},
	{Name: "pause:3.9", Digest: "sha256:7031c1b283388d2c2e09b57badb803c05ebed362dc88d84b480cc47f72a21097"},
}

var repoRoot = ""
var binDir = ""

func TestMain(m *testing.M) {
	if e := os.Getenv("REGISTRY_ENDPOINT"); e != "" {
		endpoint = e
	}
	rr, err := integration.ModuleRootDir()
	if err != nil {
		panic("failed to get root dir: " + err.Error())
	}
	repoRoot = rr
	binDir = filepath.Join(repoRoot, "bin")
	if err := os.Chdir(repoRoot); err != nil {
		panic("failed to chdir to repo root: " + err.Error())
	}
	os.Exit(m.Run())
}

// installs tool to binDir using go install
func goInstall(t *testing.T, tool string) {
	buildCmd := exec.Command("go", "install", tool)
	buildCmd.Env = append(os.Environ(), "GOBIN="+binDir)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Errorf("Failed to get %q: %v", tool, err)
		t.Error("Output:")
		t.Fatal(string(out))
	}
}

// common helper for executing test pull and checking output
func testPull(t *testing.T, tc *testCase, pullCmd *exec.Cmd) {
	out, err := pullCmd.CombinedOutput()
	if err != nil {
		t.Errorf("Failed to pull image: %q with err %v", tc.Name, err)
		t.Error("Output from command:")
		t.Fatal(string(out))
	} else if tc.Digest != "" && !strings.Contains(string(out), tc.Digest) {
		t.Error("pull output does not contain expected digest")
		t.Error("Output from command:")
		t.Fatal(string(out))
	}
}

func TestE2ECranePull(t *testing.T) {
	t.Parallel()
	// install crane
	goInstall(t, "github.com/google/go-containerregistry/cmd/crane@latest")
	// pull test images
	for i := range testCases {
		tc := &testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			pullCmd := exec.Command("./crane", "pull", "--verbose", tc.Ref(), "/dev/null")
			pullCmd.Dir = binDir
			testPull(t, tc, pullCmd)
		})
	}
}
