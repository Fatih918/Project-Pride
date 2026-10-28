package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type cacheTestProvider struct {
	closeCalls atomic.Int32
}

func (p *cacheTestProvider) Execute(context.Context, Request) (Response, error) {
	return Response{}, nil
}

func (p *cacheTestProvider) Close() error {
	p.closeCalls.Add(1)
	return nil
}

func newTestCache(t *testing.T) *ClientCache {
	t.Helper()
	cache, err := NewClientCache()
	if err != nil {
		t.Fatalf("NewClientCache: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})
	return cache
}

func newTestKey(t *testing.T, cache *ClientCache, configRef string, credential string) ClientKey {
	t.Helper()
	key, err := cache.NewKey(
		configRef,
		"gemini",
		map[string]any{"model": "test-model"},
		[]byte(credential),
	)
	if err != nil {
		t.Fatalf("NewKey: %v", err)
	}
	return key
}

func TestClientCacheReusesProviderForSameEffectiveKey(t *testing.T) {
	cache := newTestCache(t)
	key := newTestKey(t, cache, "config-a", "secret-a")
	p := &cacheTestProvider{}
	var builds atomic.Int32
	factory := func(context.Context) (Provider, error) {
		builds.Add(1)
		return p, nil
	}

	first, err := cache.Acquire(context.Background(), key, factory)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	second, err := cache.Acquire(context.Background(), key, factory)
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if first.Provider() != second.Provider() {
		t.Fatal("expected same provider instance for identical key")
	}
	if builds.Load() != 1 {
		t.Fatalf("factory calls = %d, want 1", builds.Load())
	}

	if err := first.Release(); err != nil {
		t.Fatalf("first Release: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
	if p.closeCalls.Load() != 0 {
		t.Fatal("active cached provider should not be closed before invalidation")
	}
}

func TestClientCacheKeyChangesWhenCredentialOrConfigurationChanges(t *testing.T) {
	cache := newTestCache(t)
	base := newTestKey(t, cache, "config-a", "secret-a")
	rotated := newTestKey(t, cache, "config-a", "secret-b")
	otherConfig := newTestKey(t, cache, "config-b", "secret-a")
	if base == rotated {
		t.Fatal("credential change must produce a different key")
	}
	if base == otherConfig {
		t.Fatal("different configurations must not share a key by default")
	}
}

func TestClientCacheInvalidationDefersCloseUntilLeaseReleased(t *testing.T) {
	cache := newTestCache(t)
	key := newTestKey(t, cache, "config-a", "secret-a")
	p := &cacheTestProvider{}
	lease, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
		return p, nil
	})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	if err := cache.InvalidateConfiguration("config-a"); err != nil {
		t.Fatalf("InvalidateConfiguration: %v", err)
	}
	if p.closeCalls.Load() != 0 {
		t.Fatal("provider closed while an active lease still exists")
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if p.closeCalls.Load() != 1 {
		t.Fatalf("Close calls = %d, want 1", p.closeCalls.Load())
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
	if p.closeCalls.Load() != 1 {
		t.Fatalf("Close calls after second release = %d, want 1", p.closeCalls.Load())
	}
}

func TestClientCacheCredentialRotationCreatesNewProvider(t *testing.T) {
	cache := newTestCache(t)
	oldKey := newTestKey(t, cache, "config-a", "old-secret")
	newKey := newTestKey(t, cache, "config-a", "new-secret")
	oldProvider := &cacheTestProvider{}
	newProvider := &cacheTestProvider{}

	oldLease, err := cache.Acquire(context.Background(), oldKey, func(context.Context) (Provider, error) {
		return oldProvider, nil
	})
	if err != nil {
		t.Fatalf("old Acquire: %v", err)
	}
	newLease, err := cache.Acquire(context.Background(), newKey, func(context.Context) (Provider, error) {
		return newProvider, nil
	})
	if err != nil {
		t.Fatalf("new Acquire: %v", err)
	}
	if oldLease.Provider() == newLease.Provider() {
		t.Fatal("credential rotation must not reuse the old provider")
	}

	if err := cache.InvalidateConfiguration("config-a"); err != nil {
		t.Fatalf("InvalidateConfiguration: %v", err)
	}
	if oldProvider.closeCalls.Load() != 0 || newProvider.closeCalls.Load() != 0 {
		t.Fatal("invalidation must defer close while leases are active")
	}
	_ = oldLease.Release()
	_ = newLease.Release()
	if oldProvider.closeCalls.Load() != 1 || newProvider.closeCalls.Load() != 1 {
		t.Fatal("both retired providers should close after their leases are released")
	}
}

func TestClientCacheConcurrentAcquireBuildsOnce(t *testing.T) {
	cache := newTestCache(t)
	key := newTestKey(t, cache, "config-a", "secret-a")
	p := &cacheTestProvider{}
	var builds atomic.Int32
	start := make(chan struct{})
	const workers = 16

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	leases := make(chan *ProviderLease, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			lease, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
				builds.Add(1)
				time.Sleep(10 * time.Millisecond)
				return p, nil
			})
			if err != nil {
				errs <- err
				return
			}
			leases <- lease
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(leases)

	for err := range errs {
		t.Errorf("Acquire: %v", err)
	}
	for lease := range leases {
		if lease.Provider() != p {
			t.Error("Acquire returned unexpected provider instance")
		}
		_ = lease.Release()
	}
	if builds.Load() != 1 {
		t.Fatalf("factory calls = %d, want 1", builds.Load())
	}
}

func TestClientCacheCloseRetiresActiveProvider(t *testing.T) {
	cache := newTestCache(t)
	key := newTestKey(t, cache, "config-a", "secret-a")
	p := &cacheTestProvider{}
	lease, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
		return p, nil
	})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := cache.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if p.closeCalls.Load() != 0 {
		t.Fatal("Close must defer provider close while a lease is active")
	}
	if _, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
		return &cacheTestProvider{}, nil
	}); !errors.Is(err, ErrClientCacheClosed) {
		t.Fatalf("Acquire after Close error = %v, want ErrClientCacheClosed", err)
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if p.closeCalls.Load() != 1 {
		t.Fatalf("Close calls = %d, want 1", p.closeCalls.Load())
	}
}

func TestClientCacheRejectsFactoryFailureAndCanRetry(t *testing.T) {
	cache := newTestCache(t)
	key := newTestKey(t, cache, "config-a", "secret-a")
	expected := errors.New("build failed")
	if _, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
		return nil, expected
	}); !errors.Is(err, expected) {
		t.Fatalf("Acquire error = %v, want wrapped build error", err)
	}

	p := &cacheTestProvider{}
	lease, err := cache.Acquire(context.Background(), key, func(context.Context) (Provider, error) {
		return p, nil
	})
	if err != nil {
		t.Fatalf("Acquire retry: %v", err)
	}
	_ = lease.Release()
}
