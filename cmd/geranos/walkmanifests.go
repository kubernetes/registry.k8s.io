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
	"github.com/google/go-containerregistry/pkg/v1/google"
)

type walkManifestsFunc func(ref name.Reference) error

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
func walkManifestsGCP(repo name.Repository, walkManifest walkManifestsFunc) error {
	return google.Walk(repo, func(repo name.Repository, tags *google.Tags, err error) error {
		for digest := range tags.Manifests {
			ref, err := name.ParseReference(fmt.Sprintf("%s@%s", repo, digest))
			if err != nil {
				return err
			}
			if err := walkManifest(ref); err != nil {
				return err
			}
		}
		return nil
	})
}
