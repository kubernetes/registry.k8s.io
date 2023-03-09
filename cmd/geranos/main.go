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
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"k8s.io/klog/v2"
)

func main() {
	Main()
}

// Main is the application entrypoint, which injects globals to Run
func Main() {
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

	// 60*60s = 3600 RPM, which should be well below our current 5000 RPM
	// limit, even with the host node making other registry API calls
	registryRateLimit := NewRateLimitRoundTripper(60, 1)

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
	err = WalkImageLayersGCP(registryRateLimit, repo, func(ref name.Reference, layers []v1.Layer) error {
		klog.Infof("Processing image: %s", ref.String())
		return copyImageLayers(s3Uploader, s3Bucket, layers)
	})
	if err != nil {
		return err
	}
	klog.Info("Done!")
	return nil
}

func copyImageLayers(s3Uploader *s3Uploader, bucket string, layers []v1.Layer) error {
	// copy all layers
	for _, layer := range layers {
		if err := s3Uploader.CopyToS3(bucket, layer); err != nil {
			return err
		}
	}
	return nil
}
