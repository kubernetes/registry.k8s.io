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
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"k8s.io/klog/v2"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// 40*60s = 2400 RPM, which leaves 100 overhead for list calls to reach 2500
// our actual limit currently is 5000 RPM but we're proposing lowering it ...
// TODO: unfortunately this is inherently global state, but the current approach
// is kinda hacky
var gcpRateLimiter *rate.Limiter = rate.NewLimiter(40, 1)

// WalkImageLAyersFunc is used to visit an image
type WalkImageLayersFunc func(ref name.Reference, layers []v1.Layer) error

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
func WalkImageLayersGCP(repo name.Repository, walkImageLayers WalkImageLayersFunc) error {
	g := new(errgroup.Group)
	// TODO: This is really just an approximation to avoid exceeding typical socket limits
	// See also quota limits:
	// https://cloud.google.com/artifact-registry/quotas
	g.SetLimit(10)
	g.Go(func() error {
		return google.Walk(repo, func(r name.Repository, tags *google.Tags, err error) error {
			if err != nil {
				return err
			}
			for digest, metadata := range tags.Manifests {
				digest := digest
				// google.Walk already walks the child manifests
				if metadata.MediaType == string(types.DockerManifestList) || metadata.MediaType == string(types.OCIImageIndex) {
					continue
				}
				ref, err := name.ParseReference(fmt.Sprintf("%s@%s", r, digest))
				if err != nil {
					return err
				}
				g.Go(func() error {
					return walkManifestLayers(ref, walkImageLayers)
				})
			}
			return nil
		})
	})
	return g.Wait()
}

func walkManifestLayers(ref name.Reference, walkImageLayers WalkImageLayersFunc) error {
	_ = gcpRateLimiter.Wait(context.Background())
	desc, err := remote.Get(ref)
	if err != nil {
		return err
	}

	// google.Walk already resolves these to individual manifests
	if desc.MediaType.IsIndex() {
		klog.Warningf("Skipping Index: %s", ref.String())
		return nil
	}

	// Specially handle schema 1
	// https://github.com/google/go-containerregistry/issues/377
	if desc.MediaType == types.DockerManifestSchema1 || desc.MediaType == types.DockerManifestSchema1Signed {
		layers, err := layersForV1(ref, desc)
		if err != nil {
			return err
		}
		return walkImageLayers(ref, layers)
	}

	// we don't expect anything other than index, or image ...
	if !desc.MediaType.IsImage() {
		klog.Warningf("Un-handled type: %s for %s", desc.MediaType, ref.String())
		return nil
	}

	// Handle normal images
	image, err := desc.Image()
	if err != nil {
		return err
	}
	layers, err := imageToLayers(image)
	if err != nil {
		return err
	}
	return walkImageLayers(ref, layers)
}

func imageToLayers(image v1.Image) ([]v1.Layer, error) {
	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}
	// TODO: upload and use config layers
	// for now: not uploading them matches rclone syncing from the GCR GCS
	/*
			configLayer, err := partial.ConfigLayer(image)
			if err != nil {
				return nil, err
			}
		return append(layers, configLayer), nil
	*/
	return layers, nil
}
