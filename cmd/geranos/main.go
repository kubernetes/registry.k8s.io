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
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"

	"k8s.io/klog/v2"
)

func main() {
	// one of the backing registries for registry.k8s.io
	// TODO: make configurable later
	const sourceRegistry = "us-central1-docker.pkg.dev/k8s-artifacts-prod/images"

	// TODO: make configurable later
	const s3Bucket = "s3:prod-registry-k8s-io-us-east-2"

	repo, err := name.NewRepository(sourceRegistry)
	if err != nil {
		klog.Fatal(err)
	}
	s3Uploader, err := newS3Uploader()
	if err != nil {
		klog.Fatal(err)
	}

	// copy layers from all images in the repo
	err = walkImagesGCP(repo, func(ref name.Reference, image v1.Image) error {
		klog.Infof("Processing image: %s", ref.String())
		copyImageLayers(s3Uploader, s3Bucket, image)
		return nil
	})
	if err != nil {
		klog.Fatal(err)
	} else {
		klog.Info("Done!")
	}
}

func copyImageLayers(s3Uploader *s3Uploader, bucket string, image v1.Image) error {
	// get all image blobs as v1.Layer
	layers, err := image.Layers()
	if err != nil {
		return err
	}
	configLayer, err := partial.ConfigLayer(image)
	if err != nil {
		return err
	}
	layers = append(layers, configLayer)
	// copy all layers
	for _, layer := range layers {
		if err := s3Uploader.CopyToS3(bucket, layer); err != nil {
			return err
		}
	}
	return nil
}
