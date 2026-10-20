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

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

// maxRepositoryListBytes bounds how much of the repository listing response
// we are willing to read
const maxRepositoryListBytes = 8 << 20

// ListUpstreamRepositories returns the set of repositories hosted under the
// configured upstream registry path, e.g.
// https://us-central1-docker.pkg.dev/v2/k8s-artifacts-prod/images/tags/list
//
// It relies on the `child` field GCR and Artifact Registry add to the tags
// list response. Only immediate children are listed, so nested images like
// sig-storage/csi-provisioner are represented by their sig-storage parent.
//
// See KnownRepositories for keeping the result up to date.
func ListUpstreamRepositories(rc RegistryConfig) (map[string]struct{}, error) {
	listURL := rc.UpstreamRegistryEndpoint + path.Join("/v2/", rc.UpstreamRegistryPath, "tags/list")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(listURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d listing repositories at %s", resp.StatusCode, listURL)
	}
	var listing struct {
		Child []string `json:"child"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRepositoryListBytes)).Decode(&listing); err != nil {
		return nil, err
	}
	if len(listing.Child) == 0 {
		return nil, errors.New("upstream registry listed no repositories")
	}
	repositories := make(map[string]struct{}, len(listing.Child))
	for _, child := range listing.Child {
		repositories[child] = struct{}{}
	}
	return repositories, nil
}

// KnownRepositories is the concurrency safe, refreshable set of upstream
// repositories, the zero value is an empty set
type KnownRepositories struct {
	repositories atomic.Pointer[map[string]struct{}]
}

// Get returns the current set, nil if never successfully refreshed
func (k *KnownRepositories) Get() map[string]struct{} {
	if k == nil {
		return nil
	}
	if repositories := k.repositories.Load(); repositories != nil {
		return *repositories
	}
	return nil
}

// Refresh replaces the set with ListUpstreamRepositories, keeping the
// previous set on failure
func (k *KnownRepositories) Refresh(rc RegistryConfig) error {
	repositories, err := ListUpstreamRepositories(rc)
	if err != nil {
		return err
	}
	k.repositories.Store(&repositories)
	return nil
}

// topLevelRepository returns the first segment of the image name in a
// /v2/<name>/... request path, or "" if there is none
func topLevelRepository(requestPath string) string {
	rest, ok := strings.CutPrefix(requestPath, "/v2/")
	if !ok {
		return ""
	}
	repository, _, _ := strings.Cut(rest, "/")
	return repository
}
