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
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeHead struct {
	mu     sync.Mutex
	calls  int
	status int
	err    error
}

func (f *fakeHead) head(_, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.status, f.err
}

func (f *fakeHead) set(status int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.err = status, err
}

func (f *fakeHead) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestUpstreamChecker(h *fakeHead, now *time.Time) *cachedUpstreamChecker {
	c := newCachedUpstreamChecker()
	c.head = h.head
	c.now = func() time.Time { return *now }
	return c
}

const testContentURL = "https://europe-west1-docker.pkg.dev/v2/pause/manifests/sha256:04d3ed4d"

func TestUpstreamCheckerFoundIsCached(t *testing.T) {
	now := time.Now()
	h := &fakeHead{status: http.StatusOK}
	c := newTestUpstreamChecker(h, &now)
	for range 3 {
		if c.Missing(testContentURL, "") {
			t.Fatal("expected content to be found")
		}
	}
	now = now.Add(24 * time.Hour)
	if c.Missing(testContentURL, "") {
		t.Fatal("expected content to be found")
	}
	if calls := h.callCount(); calls != 1 {
		t.Fatalf("expected 1 check, got: %d", calls)
	}
}

func TestUpstreamCheckerMissingIsCachedWithTTL(t *testing.T) {
	now := time.Now()
	h := &fakeHead{status: http.StatusNotFound}
	c := newTestUpstreamChecker(h, &now)
	for range 3 {
		if !c.Missing(testContentURL, "") {
			t.Fatal("expected content to be missing")
		}
	}
	if calls := h.callCount(); calls != 1 {
		t.Fatalf("expected 1 check, got: %d", calls)
	}
	// content replicated to the regional upstream after the TTL
	now = now.Add(upstreamMissingTTL)
	h.set(http.StatusOK, nil)
	if c.Missing(testContentURL, "") {
		t.Fatal("expected content to be found after the TTL")
	}
	if calls := h.callCount(); calls != 2 {
		t.Fatalf("expected 2 checks, got: %d", calls)
	}
}

func TestUpstreamCheckerMissingIsBounded(t *testing.T) {
	now := time.Now()
	h := &fakeHead{status: http.StatusNotFound}
	c := newTestUpstreamChecker(h, &now)
	for i := range upstreamMissingMaxEntries + 1 {
		c.Missing(fmt.Sprintf("%s%d", testContentURL, i), "")
	}
	if n := len(c.missing); n != upstreamMissingMaxEntries {
		t.Fatalf("expected %d cached entries, got: %d", upstreamMissingMaxEntries, n)
	}
	// the full cache is not swept again before its earliest entry expired
	c.missing[testContentURL+"expired"] = now
	now = now.Add(upstreamMissingTTL - time.Second)
	c.Missing(testContentURL+"new", "")
	if _, cached := c.missing[testContentURL+"expired"]; !cached {
		t.Fatal("expected no sweep before the earliest entry expired")
	}
	// expired entries make room for new ones
	now = now.Add(time.Second)
	if !c.Missing(testContentURL, "") {
		t.Fatal("expected content to be missing")
	}
	if n := len(c.missing); n != 1 {
		t.Fatalf("expected expired entries to be removed, got: %d entries", n)
	}
}

func TestUpstreamCheckerMissingSweepsAtEarliestExpiry(t *testing.T) {
	start := time.Now()
	now := start
	h := &fakeHead{status: http.StatusNotFound}
	c := newTestUpstreamChecker(h, &now)
	c.Missing(testContentURL+"first", "")
	now = now.Add(time.Minute)
	for i := range upstreamMissingMaxEntries - 1 {
		c.Missing(fmt.Sprintf("%s%d", testContentURL, i), "")
	}
	// the full cache has no expired entries, the next sweep is scheduled
	// for the earliest entry
	c.Missing(testContentURL+"dropped", "")
	if _, cached := c.missing[testContentURL+"dropped"]; cached {
		t.Fatal("expected no room in a full cache without expired entries")
	}
	now = start.Add(upstreamMissingTTL)
	if !c.Missing(testContentURL+"new", "") {
		t.Fatal("expected content to be missing")
	}
	if _, cached := c.missing[testContentURL+"first"]; cached {
		t.Fatal("expected the earliest entry to be removed")
	}
	if _, cached := c.missing[testContentURL+"new"]; !cached {
		t.Fatal("expected new entry to be cached")
	}
}

func TestUpstreamCheckerFailuresAreNotMissing(t *testing.T) {
	for _, tc := range []struct {
		Name   string
		Status int
		Err    error
	}{
		{Name: "error", Err: errors.New("timeout")},
		{Name: "server error", Status: http.StatusServiceUnavailable},
		{Name: "rate limited", Status: http.StatusTooManyRequests},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			now := time.Now()
			h := &fakeHead{status: tc.Status, err: tc.Err}
			c := newTestUpstreamChecker(h, &now)
			for range upstreamFailureThreshold - 1 {
				if c.Missing(testContentURL, "") {
					t.Fatal("expected failed check to not report missing content")
				}
			}
			if calls := h.callCount(); calls != upstreamFailureThreshold-1 {
				t.Fatalf("expected failed checks to not be cached, got %d checks", calls)
			}
		})
	}
}

func TestUpstreamCheckerCircuitBreaker(t *testing.T) {
	now := time.Now()
	h := &fakeHead{err: errors.New("timeout")}
	c := newTestUpstreamChecker(h, &now)
	for range upstreamFailureThreshold * 2 {
		if c.Missing(testContentURL, "") {
			t.Fatal("expected failed check to not report missing content")
		}
	}
	if calls := h.callCount(); calls != upstreamFailureThreshold {
		t.Fatalf("expected checks to stop after %d failures, got: %d", upstreamFailureThreshold, calls)
	}
	// checks resume after the cooldown
	now = now.Add(upstreamBreakerCooldown)
	h.set(http.StatusNotFound, nil)
	if !c.Missing(testContentURL, "") {
		t.Fatal("expected content to be missing after the cooldown")
	}
	if calls := h.callCount(); calls != upstreamFailureThreshold+1 {
		t.Fatalf("expected checks to resume, got: %d", calls)
	}
}

func TestUpstreamCheckerClientErrorsDoNotTripBreaker(t *testing.T) {
	now := time.Now()
	h := &fakeHead{status: http.StatusBadRequest}
	c := newTestUpstreamChecker(h, &now)
	for range upstreamFailureThreshold * 2 {
		if c.Missing(testContentURL, "") {
			t.Fatal("expected rejected check to not report missing content")
		}
	}
	if calls := h.callCount(); calls != upstreamFailureThreshold*2 {
		t.Fatalf("expected every rejected check to reach the upstream, got: %d", calls)
	}
	// a real failure after them still needs the full threshold
	h.set(0, errors.New("timeout"))
	for range upstreamFailureThreshold - 1 {
		c.Missing(testContentURL, "")
	}
	h.set(http.StatusNotFound, nil)
	if !c.Missing(testContentURL, "") {
		t.Fatal("expected checks to continue below the failure threshold")
	}
}

func TestUpstreamCheckerDeduplicatesConcurrentChecks(t *testing.T) {
	release := make(chan struct{})
	var calls atomic.Int32
	c := newCachedUpstreamChecker()
	c.head = func(_, _ string) (int, error) {
		calls.Add(1)
		<-release
		return http.StatusNotFound, nil
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if !c.Missing(testContentURL, "") {
				t.Error("expected content to be missing")
			}
		})
	}
	// give the goroutines a chance to wait on the same check
	time.Sleep(10 * time.Millisecond)
	close(release)
	wg.Wait()
	// late callers hit the cache, so there is exactly one check either way
	if n := calls.Load(); n != 1 {
		t.Fatalf("expected 1 check, got: %d", n)
	}
}
