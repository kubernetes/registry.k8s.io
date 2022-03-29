package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
)

var (
	defaultUpstreamRegistry = "registry.k8s.io"
	defaultUserAgent        = "sigs.k8s.io/oci-proxy/test"
	pauseImage              = "pause:3.6"
	craneOptions            = crane.WithUserAgent(defaultUserAgent)
	vendorRegistries        = []string{"k8s.gcr.io", "some-url.aws"}

	testOptions = &TestOptions{
		upstreamRegistry: envOrDefault("UPSTREAM_REGISTRY", defaultUpstreamRegistry),
	}
)

type TestOptions struct {
	upstreamRegistry string
}

func envOrDefault(env, defaultValue string) (output string) {
	output = os.Getenv(env)
	if output == "" {
		output = defaultValue
	}
	return output
}

func TestPullPause(t *testing.T) {
	image, err := crane.Pull(testOptions.upstreamRegistry+"/"+pauseImage, craneOptions)
	if err != nil {
		t.Errorf("Failed to pull image, %v", err)
	}
	manifest, err := image.Manifest()
	if err != nil {
		t.Errorf("Failed to get image manifest, %v", err)
	}
	if len(manifest.Layers) == 0 {
		t.Errorf("No layers found in manifest (%v)", testOptions.upstreamRegistry+"/"+pauseImage)
	}
	for _, l := range manifest.Layers {
		layerRef := fmt.Sprintf("%v/%v@%v:%v", testOptions.upstreamRegistry, pauseImage, l.Digest.Algorithm, l.Digest.Hex)
		layer, err := crane.PullLayer(layerRef, craneOptions)
		if err != nil {
			t.Errorf("Failed to get image manifest, %v", err)
		}
		size, err := layer.Size()
		if err != nil {
			t.Errorf("Failed to get image layer size, %v", err)
		}
		if size == 0 {
			t.Errorf("Size should not be zero for layer (%v)", l.Digest.Hex)
		}
	}
}

func TestImageWontExist(t *testing.T) {
	_, err := crane.Pull(testOptions.upstreamRegistry+"/aaaaaaaaaaaaaa", craneOptions)
	if err == nil {
		t.Errorf("This image should not exist, %v", err)
	}
}

func TestResolveV2(t *testing.T) {
	client := http.Client{}
	requestPath := "/v2/"
	requestURL := fmt.Sprintf("https://%v%v", testOptions.upstreamRegistry, requestPath)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		t.Errorf("Error requesting fake backend, %v", err)
	}
	req.Header.Add("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Error requesting fake backend, %v", err)
	}
	fmt.Printf("%v\n", resp.Request.URL.RawPath)
	hostMatchesOneVendorRegistry := false
	pathMatchesOneVendorRegistry := false
	vendorRegistriesWithSchemeAndPath := []string{}
	for _, vr := range vendorRegistries {
		vendorRegistriesWithSchemeAndPath = append(vendorRegistriesWithSchemeAndPath, fmt.Sprintf("https://%v%v", vr, requestPath))
		if resp.Request.URL.Host == vr {
			hostMatchesOneVendorRegistry = true
		}
		if resp.Request.URL.Path == requestPath {
			pathMatchesOneVendorRegistry = true
		}
	}
	if hostMatchesOneVendorRegistry == false {
		t.Errorf("Expected host (%v) to resolve to one of '%v', instead resolved to %v", requestURL, strings.Join(vendorRegistries, ", "), resp.Request.URL.Host)
	}
	if pathMatchesOneVendorRegistry == false {
		t.Errorf("Expected url (%v) to resolve to one of '%v', instead resolved to '%v//%v%v'", requestURL, strings.Join(vendorRegistriesWithSchemeAndPath, ", "), resp.Request.URL.Scheme, resp.Request.URL.Host, resp.Request.URL.Path)
	}
}

func TestV2(t *testing.T) {
	client := http.Client{}
	requestPath := "/v2/"
	requestURL := fmt.Sprintf("https://%v%v", testOptions.upstreamRegistry, requestPath)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		t.Errorf("Error requesting fake backend, %v", err)
	}
	req.Header.Add("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Error requesting fake backend, %v", err)
	}
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Errorf("Expected response code (%v) to be %v", resp.StatusCode, http.StatusPermanentRedirect)
	}
}
