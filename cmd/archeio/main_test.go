//go:build !nointegration

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
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/validate"

	"k8s.io/registry.k8s.io/internal/integration"
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs"
)

type integrationTestCase struct {
	Name   string
	FakeIP string
	Image  string
	Digest string
}

// TestIntegrationMain tests the entire, built binary with an integration
// test, pulling images with crane
func TestIntegrationMain(t *testing.T) {
	// setup crane
	rootDir, err := integration.ModuleRootDir()
	if err != nil {
		t.Fatalf("Failed to detect module root dir: %v", err)
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
	serverCmd := exec.Command("./archeio", "-v=9")
	serverCmd.Dir = filepath.Join(rootDir, "bin")
	serverCmd.Env = append(serverCmd.Env, "PORT="+testPort)
	serverCmd.Stderr = os.Stderr
	go func() {
		serverErrChan <- serverCmd.Start()
		serverErrChan <- serverCmd.Wait()
	}()
	t.Cleanup(func() {
		if err := serverCmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to signal archeio: %v", err)
		}
		if err := <-serverErrChan; err != nil {
			t.Fatalf("archeio did not exit cleanly: %v", err)
		}
	})

	// wait for server to be up and running
	startErr := <-serverErrChan
	if startErr != nil {
		t.Fatalf("Failed to start archeio: %v", err)
	}
	if !tryUntil(time.Now().Add(time.Second), func() bool {
		_, err := http.Get("http://" + testAddr + "/v2/")
		return err == nil
	}) {
		t.Fatal("timed out waiting for archeio to be ready")
	}

	// perform many test pulls ...
	testCases := makeTestCases(t)
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ref := testAddr + "/" + tc.Image
			// ensure we supply fake IP info from test case
			craneOpts := []crane.Option{crane.WithTransport(newFakeIPTransport(tc.FakeIP))}
			// test fetching digest first
			digest, err := crane.Digest(ref, craneOpts...)
			if err != nil {
				t.Errorf("Fetch digest for %q failed: %v", ref, err)
			}
			if digest != tc.Digest {
				t.Errorf("Wrong digest for %q", ref)
				t.Errorf("Received: %q", digest)
				t.Errorf("Expected: %q", tc.Digest)
			}
			err = pull(ref, craneOpts...)
			if err != nil {
				t.Errorf("Pull for %q failed: %v", ref, err)
			}
		})
	}
}

func makeTestCases(t *testing.T) []integrationTestCase {
	// a few small images that we really should be able to pull
	wellKnownImages := []struct {
		Name   string
		Digest string
	}{
		{
			Name:   "pause:3.1",
			Digest: "sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea",
		},
		{
			Name:   "pause:3.9",
			Digest: "sha256:7031c1b283388d2c2e09b57badb803c05ebed362dc88d84b480cc47f72a21097",
		},
	}

	// collect interesting IPs after checking that they meet expectations
	type interestingIP struct {
		Name string
		IP   string
	}
	interestingIPs := []interestingIP{}
	cidrs := cloudcidrs.NewIPMapper()

	// One for GCP because we host there and have code paths for this
	const gcpIP = "35.220.26.1"
	if info, matches := cidrs.GetIP(netip.MustParseAddr(gcpIP)); !matches || info.Cloud != cloudcidrs.GCP {
		t.Fatalf("Expected %q to be a GCP IP but is not detected as one with current data", gcpIP)
	}
	interestingIPs = append(interestingIPs, interestingIP{Name: "GCP", IP: gcpIP})

	// One for AWS because we host there and have code paths for this
	const awsIP = "35.180.1.1"
	if info, matches := cidrs.GetIP(netip.MustParseAddr(awsIP)); !matches || info.Cloud != cloudcidrs.AWS {
		t.Fatalf("Expected %q to be an AWS IP but is not detected as one with current data", awsIP)
	}
	interestingIPs = append(interestingIPs, interestingIP{Name: "AWS", IP: awsIP})

	// we obviously won't see this in the wild, but we also know
	// it should not match GCP, AWS or any future providers
	const externalIP = "192.168.0.1"
	if _, matches := cidrs.GetIP(netip.MustParseAddr(externalIP)); matches {
		t.Fatalf("Expected %q to not match any provider IP range but it dies", externalIP)
	}
	interestingIPs = append(interestingIPs, interestingIP{Name: "External", IP: externalIP})

	// generate testcases from test data, for every interesting IP pull each image
	testCases := []integrationTestCase{}
	for _, image := range wellKnownImages {
		for _, ip := range interestingIPs {
			testCases = append(testCases, integrationTestCase{
				Name:   fmt.Sprintf("IP:%s (%q),Image:%q", ip.Name, ip.IP, image.Name),
				FakeIP: ip.IP,
				Image:  image.Name,
				Digest: image.Digest,
			})
		}
	}
	return testCases
}

func pull(image string, options ...crane.Option) error {
	img, err := crane.Pull(image, options...)
	if err != nil {
		return err
	}
	return validate.Image(img)
}

type fakeIPTransport struct {
	fakeXForwardFor string
	h               http.RoundTripper
}

var _ http.RoundTripper = &fakeIPTransport{}

func (f *fakeIPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("X-Forwarded-For", f.fakeXForwardFor)
	return f.h.RoundTrip(r)
}

func newFakeIPTransport(fakeIP string) *fakeIPTransport {
	return &fakeIPTransport{
		fakeXForwardFor: fakeIP + ",0.0.0.0",
		h:               http.DefaultTransport,
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
