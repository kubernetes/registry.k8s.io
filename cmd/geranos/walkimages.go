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

package main

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"k8s.io/klog/v2"
)

// WalkImageFunc is used to visit an image
type WalkImageFunc func(ref name.Reference, image v1.Image) error

// Unfortunately this is only doable on GCP currently.
//
// TODO: To support other registries in the meantime, we could require a list of
// image names as an input and plumb that through, then list tags and get something
// close to this. The _catalog endpoint + tag listing could also work in some cases.
//
// However, even then, this is more complete because it lists all manifests, not just tags.
// It's also simpler and more efficient.
//
// See: https://github.com/opencontainers/distribution-spec/issues/222
func WalkImagesGCP(repo name.Repository, walkImage WalkImageFunc) error {
	return google.Walk(repo, func(repo name.Repository, tags *google.Tags, err error) error {
		for digest := range tags.Manifests {
			ref, err := name.ParseReference(fmt.Sprintf("%s@%s", repo, digest))
			if err != nil {
				return err
			}
			if err := walkManifestImage(ref, walkImage); err != nil {
				return err
			}
		}
		return nil
	})
}

func walkManifestImage(ref name.Reference, walkImage WalkImageFunc) error {
	desc, err := remote.Get(ref)
	if err != nil {
		return err
	}

	// google.Walk already resolves these to individual manifests
	if desc.MediaType.IsIndex() {
		return nil
	}

	// It's unlikely we need to support these
	// https://github.com/google/go-containerregistry/issues/377
	if desc.MediaType == types.DockerManifestSchema1 || desc.MediaType == types.DockerManifestSchema1Signed {
		klog.Warning("Skipping ancient schema 1 image: " + ref.String())
		return nil
	}

	// we don't expect anything other than index, or image ...
	if !desc.MediaType.IsImage() {
		klog.Warningf("Un-handled type: %s", desc.MediaType)
		return nil
	}

	image, err := desc.Image()
	if err != nil {
		return err
	}
	return walkImage(ref, image)
}
