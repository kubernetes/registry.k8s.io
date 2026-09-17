/*
Copyright 2026 The Kubernetes Authors.

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
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"k8s.io/klog/v2"
)

const (
	// upstreamMissingTTL is how long content missing in the regional
	// upstream is served from the signature upstream without checking
	// again, content may still be replicated to the regional upstream
	upstreamMissingTTL = 10 * time.Minute
	// upstreamMissingMaxEntries bounds the negative cache, clients can
	// request arbitrary digests
	upstreamMissingMaxEntries = 10000
	// upstreamFailureThreshold is the number of consecutive failed checks
	// after which checks are skipped for upstreamBreakerCooldown
	upstreamFailureThreshold = 5
	upstreamBreakerCooldown  = 30 * time.Second
)

// upstreamChecker checks if content addressed requests can be served by the
// regional upstream registry
type upstreamChecker interface {
	// Missing reports whether the regional upstream answered that
	// contentURL does not exist. Failed checks report false, so that the
	// request is served by the regional upstream as without a check.
	// traceID is propagated on the outbound probe for log correlation and
	// must not affect caching
	Missing(contentURL, traceID string) bool
}

// cachedUpstreamChecker checks with an HTTP HEAD request if content exists in
// the regional upstream. Found content is immutable and cached forever,
// missing content is cached for upstreamMissingTTL. Concurrent checks for the
// same content are deduplicated, and consecutive failures skip checks for a
// while so that regional upstream outages don't stall requests.
type cachedUpstreamChecker struct {
	found blobCache
	group singleflight.Group

	mu        sync.Mutex
	missing   map[string]time.Time
	sweepAt   time.Time
	failures  int
	openUntil time.Time

	// injectable for testing
	now  func() time.Time
	head func(url, traceID string) (int, error)
}

func newCachedUpstreamChecker() *cachedUpstreamChecker {
	return &cachedUpstreamChecker{
		missing: map[string]time.Time{},
		now:     time.Now,
		head:    headStatus,
	}
}

func (c *cachedUpstreamChecker) Missing(contentURL, traceID string) bool {
	if c.found.Get(contentURL) {
		return false
	}
	now := c.now()
	c.mu.Lock()
	expiry, cached := c.missing[contentURL]
	open := now.Before(c.openUntil)
	c.mu.Unlock()
	if cached && now.Before(expiry) {
		klog.V(3).InfoS("content found in upstream missing cache", "url", contentURL, "traceID", traceID)
		return true
	}
	if open {
		klog.V(3).InfoS("skipping upstream check, recent checks failed", "url", contentURL, "traceID", traceID)
		return false
	}
	missing, _, _ := c.group.Do(contentURL, func() (any, error) {
		return c.check(contentURL, traceID), nil
	})
	return missing.(bool)
}

func (c *cachedUpstreamChecker) check(contentURL, traceID string) bool {
	klog.V(3).InfoS("content not yet in upstream existence cache; checking remote", "url", contentURL, "traceID", traceID)
	status, err := c.head(contentURL, traceID)
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case err == nil && status == http.StatusOK:
		c.failures = 0
		c.found.Put(contentURL)
		delete(c.missing, contentURL)
		return false
	case err == nil && status == http.StatusNotFound:
		c.failures = 0
		c.putMissing(contentURL, now)
		return true
	}
	klog.V(3).InfoS("failed to check upstream content", "url", contentURL, "status", status, "err", err, "traceID", traceID)
	c.failures++
	if c.failures >= upstreamFailureThreshold {
		klog.InfoS("skipping upstream checks after consecutive failures", "failures", c.failures, "cooldown", upstreamBreakerCooldown)
		c.failures = 0
		c.openUntil = now.Add(upstreamBreakerCooldown)
	}
	return false
}

// putMissing caches contentURL as missing, c.mu must be held
func (c *cachedUpstreamChecker) putMissing(contentURL string, now time.Time) {
	if len(c.missing) >= upstreamMissingMaxEntries {
		// a full cache is only swept once its earliest entry expired
		if now.Before(c.sweepAt) {
			return
		}
		c.sweepAt = now.Add(upstreamMissingTTL)
		for u, expiry := range c.missing {
			if !now.Before(expiry) {
				delete(c.missing, u)
			} else if expiry.Before(c.sweepAt) {
				c.sweepAt = expiry
			}
		}
		if len(c.missing) >= upstreamMissingMaxEntries {
			return
		}
	}
	c.missing[contentURL] = now.Add(upstreamMissingTTL)
}
