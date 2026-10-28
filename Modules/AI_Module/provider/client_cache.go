package provider

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
)

var (
	ErrClientCacheClosed      = errors.New("provider client cache is closed")
	ErrClientCacheInvalidated = errors.New("provider client cache entry was invalidated during creation")
	ErrInvalidClientKey       = errors.New("provider client cache key is invalid")
)

type ClientFactory func(context.Context) (Provider, error)

// ClientKey is an opaque, cache-scoped identity for one configuration's
// effective provider client. Its fingerprint never contains the raw credential.
type ClientKey struct {
	configurationRef string
	digest            string
}

// ClientCache owns provider clients and defers closing retired clients until
// every active lease has been released.
type ClientCache struct {
	mu             sync.Mutex
	entries        map[ClientKey]*clientCacheEntry
	fingerprintKey []byte
	closed         bool
}

type clientCacheEntry struct {
	key       ClientKey
	ready     chan struct{}
	provider  Provider
	err       error
	refs      int
	building  bool
	retired   bool
	closeDone bool
}

type ProviderLease struct {
	cache      *ClientCache
	entry      *clientCacheEntry
	provider   Provider
	once       sync.Once
	releaseErr error
}

// NewClientCache creates a cache with a random, process-local HMAC key used to
// fingerprint effective configuration and credential values.
func NewClientCache() (*ClientCache, error) {
	fingerprintKey := make([]byte, 32)
	if _, err := rand.Read(fingerprintKey); err != nil {
		return nil, fmt.Errorf("create provider cache fingerprint key: %w", err)
	}

	return &ClientCache{
		entries:        make(map[ClientKey]*clientCacheEntry),
		fingerprintKey: fingerprintKey,
	}, nil
}

// NewKey fingerprints all inputs that can change the effective provider client.
// Configuration references are included to prevent cross-configuration sharing
// by default, even when two configurations use the same credential.
func (c *ClientCache) NewKey(
	configurationRef string,
	implementationRef string,
	connectionParameters map[string]any,
	credential []byte,
) (ClientKey, error) {
	if c == nil {
		return ClientKey{}, errors.New("provider client cache must not be nil")
	}
	if configurationRef == "" || implementationRef == "" {
		return ClientKey{}, ErrInvalidClientKey
	}

	parametersJSON, err := json.Marshal(connectionParameters)
	if err != nil {
		return ClientKey{}, fmt.Errorf("encode provider connection parameters: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ClientKey{}, ErrClientCacheClosed
	}

	mac := hmac.New(sha256.New, c.fingerprintKey)
	writeFingerprintField(mac, []byte(configurationRef))
	writeFingerprintField(mac, []byte(implementationRef))
	writeFingerprintField(mac, parametersJSON)
	writeFingerprintField(mac, credential)

	return ClientKey{
		configurationRef: configurationRef,
		digest:           hex.EncodeToString(mac.Sum(nil)),
	}, nil
}

type fingerprintWriter interface {
	Write([]byte) (int, error)
}

func writeFingerprintField(w fingerprintWriter, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = w.Write(length[:])
	_, _ = w.Write(value)
}

// Acquire returns a lease to a cached provider, building it once when needed.
// The caller must release every successful lease.
func (c *ClientCache) Acquire(
	ctx context.Context,
	key ClientKey,
	factory ClientFactory,
) (*ProviderLease, error) {
	if c == nil {
		return nil, errors.New("provider client cache must not be nil")
	}
	if ctx == nil {
		return nil, errors.New("provider client cache context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if key.configurationRef == "" || key.digest == "" {
		return nil, ErrInvalidClientKey
	}
	if factory == nil {
		return nil, errors.New("provider client factory must not be nil")
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrClientCacheClosed
	}
	if entry, ok := c.entries[key]; ok {
		if !entry.building && entry.err == nil && !entry.retired {
			entry.refs++
			p := entry.provider
			c.mu.Unlock()
			return &ProviderLease{cache: c, entry: entry, provider: p}, nil
		}
		ready := entry.ready
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ready:
		}

		c.mu.Lock()
		if entry.err != nil {
			err := entry.err
			c.mu.Unlock()
			return nil, err
		}
		if c.closed {
			c.mu.Unlock()
			return nil, ErrClientCacheClosed
		}
		if entry.retired || c.entries[key] != entry {
			c.mu.Unlock()
			return nil, ErrClientCacheInvalidated
		}
		entry.refs++
		p := entry.provider
		c.mu.Unlock()
		return &ProviderLease{cache: c, entry: entry, provider: p}, nil
	}

	entry := &clientCacheEntry{
		key:      key,
		ready:    make(chan struct{}),
		building: true,
	}
	c.entries[key] = entry
	c.mu.Unlock()

	p, buildErr := factory(ctx)
	if buildErr == nil && isNilProvider(p) {
		buildErr = errors.New("provider client factory returned nil provider")
	}

	c.mu.Lock()
	entry.building = false
	entry.provider = p
	entry.err = buildErr

	if buildErr != nil || c.closed || entry.retired {
		if buildErr == nil {
			if c.closed {
				entry.err = ErrClientCacheClosed
			} else {
				entry.err = ErrClientCacheInvalidated
			}
		}
		entry.retired = true
		if c.entries[key] == entry {
			delete(c.entries, key)
		}
		close(entry.ready)
		toClose := c.takeCloseLocked(entry)
		err := entry.err
		c.mu.Unlock()

		closeErr := closeProvider(toClose)
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, err
	}

	entry.refs = 1
	close(entry.ready)
	c.mu.Unlock()

	return &ProviderLease{cache: c, entry: entry, provider: p}, nil
}

func (l *ProviderLease) Provider() Provider {
	if l == nil {
		return nil
	}
	return l.provider
}

// Release relinquishes one active use. Releasing more than once is safe.
func (l *ProviderLease) Release() error {
	if l == nil {
		return nil
	}

	l.once.Do(func() {
		if l.cache == nil || l.entry == nil {
			return
		}

		l.cache.mu.Lock()
		if l.entry.refs > 0 {
			l.entry.refs--
		}
		toClose := l.cache.takeCloseLocked(l.entry)
		l.cache.mu.Unlock()

		l.releaseErr = closeProvider(toClose)
	})
	return l.releaseErr
}

// InvalidateConfiguration retires all cached clients for a configuration.
// Active leases remain usable; closable providers are closed after the final
// lease is released. A client being built during invalidation is discarded.
func (c *ClientCache) InvalidateConfiguration(configurationRef string) error {
	if c == nil {
		return errors.New("provider client cache must not be nil")
	}
	if configurationRef == "" {
		return ErrInvalidClientKey
	}

	c.mu.Lock()
	var toClose []Provider
	for key, entry := range c.entries {
		if key.configurationRef != configurationRef {
			continue
		}
		delete(c.entries, key)
		entry.retired = true
		if p := c.takeCloseLocked(entry); p != nil {
			toClose = append(toClose, p)
		}
	}
	c.mu.Unlock()

	return closeProviders(toClose)
}

// Close prevents new acquisitions and retires every cached provider. Providers
// with active leases are closed when those leases are released.
func (c *ClientCache) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true

	var toClose []Provider
	for key, entry := range c.entries {
		delete(c.entries, key)
		entry.retired = true
		if p := c.takeCloseLocked(entry); p != nil {
			toClose = append(toClose, p)
		}
	}
	for i := range c.fingerprintKey {
		c.fingerprintKey[i] = 0
	}
	c.mu.Unlock()

	return closeProviders(toClose)
}

func (c *ClientCache) takeCloseLocked(entry *clientCacheEntry) Provider {
	if entry == nil || !entry.retired || entry.building || entry.refs != 0 || entry.closeDone {
		return nil
	}
	entry.closeDone = true
	return entry.provider
}

func closeProviders(providers []Provider) error {
	var result error
	for _, p := range providers {
		result = errors.Join(result, closeProvider(p))
	}
	return result
}

func closeProvider(p Provider) error {
	if isNilProvider(p) {
		return nil
	}
	closer, ok := p.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}

func isNilProvider(p Provider) bool {
	if p == nil {
		return true
	}
	value := reflect.ValueOf(p)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

