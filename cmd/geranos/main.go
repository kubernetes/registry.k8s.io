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
	"flag"
	"os"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"k8s.io/klog/v2"
)

func main() {
	Main()
}

// Main is the application entrypoint, which injects globals to Run
func Main() {
	klog.InitFlags(flag.CommandLine)
	flag.Parse()
	if err := Run(os.Args); err != nil {
		klog.Fatal(err)
	}
}

// Run implements the actual application logic, accepting global inputs
func Run(_ []string) error {
	// one of the backing registries for registry.k8s.io
	// TODO: make configurable later
	const sourceRegistry = "us-central1-docker.pkg.dev/k8s-artifacts-prod/images"

	// TODO: make configurable later
	const s3Bucket = "prod-registry-k8s-io-us-east-2"

	// 80*60s = 4800 RPM, below our current 5000 RPM per-user limit on the registry
	// Even with the host node making other registry API calls
	registryRateLimit := NewRateLimitRoundTripper(80)

	repo, err := name.NewRepository(sourceRegistry)
	if err != nil {
		return err
	}
	s3Uploader, err := newS3Uploader(os.Getenv("REALLY_UPLOAD") == "")
	if err != nil {
		return err
	}

	// copy layers from all images in the repo
	// TODO: print some progress logs at lower frequency instead of logging each image
	// We will punt this temporarily, as we're about to refactor how this works anyhow
	// to avoid fetching manifests for images we've already uploaded
	err = WalkImageLayersGCP(registryRateLimit, repo,
		func(ref name.Reference, layers []v1.Layer) error {
			klog.Infof("Processing image: %s", ref.String())
			return s3Uploader.UploadImage(s3Bucket, ref, layers, crane.WithTransport(registryRateLimit))
		},
		func(imageHash string) bool {
			s, _ := s3Uploader.ImageAlreadyUploaded(s3Bucket, imageHash)
			return s
		})
	if err == nil {
		klog.Info("Done!")
	}
	return err
}
