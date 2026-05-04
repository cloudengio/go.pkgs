// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cachefs provides a caching and related wrappers for ReadFileFS implementations.
package cachefs

import (
	"bytes"
	"context"
	"sync"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/sync/ctxsync"
)

type cacheEntry struct {
	data    []byte
	expires time.Time
}

// CachingReadFileFS implements a caching layer over a ReadFileFS that is
// suitable for a small numbers of small files that can be readily kept in
// memory.
type CachingReadFileFS struct {
	fs   file.ReadFileFS
	stop chan struct{}
	opts options

	mu     sync.RWMutex
	cache  map[string]cacheEntry
	wg     ctxsync.WaitGroup
	closed bool
	sf     ctxsync.SingleFlight
}

type options struct {
	ttl             time.Duration
	cleanupInterval time.Duration
	singleFlight    bool
}

type Option func(*options)

const (
	DefaultTTL             = 24 * time.Hour
	DefaultCleanupInterval = 1 * time.Hour
	DefaultSingleFlight    = false
)

// WithTTL specifies the time-to-live for cache entries. The default is DefaultTTL.
func WithTTL(d time.Duration) Option {
	return func(o *options) {
		o.ttl = d
	}
}

// WithCleanupInterval specifies the interval for periodic background cleanup of
// expired entries. The default is DefaultCleanupInterval. A value of 0 disables
// periodic cleanup, with expired entries being overwritten on access.
func WithCleanupInterval(d time.Duration) Option {
	return func(o *options) {
		o.cleanupInterval = d
	}
}

// WithSingleFlight enables single-flight behavior for concurrent calls to
// ReadFileCtx with the same name. The default is false.
func WithSingleFlight(v bool) Option {
	return func(o *options) {
		o.singleFlight = v
	}
}

// NewCachingReadFileFS creates a new CachingReadFileFS with the specified TTL
// and cleanup interval. It starts a background goroutine to periodically clear
// out expired cache entries. Call Stop to stop the background goroutine.
func NewCachingReadFileFS(fs file.ReadFileFS, opts ...Option) *CachingReadFileFS {
	o := options{
		ttl:             DefaultTTL,
		cleanupInterval: DefaultCleanupInterval,
		singleFlight:    DefaultSingleFlight,
	}
	for _, fn := range opts {
		fn(&o)
	}
	c := &CachingReadFileFS{
		fs:    fs,
		opts:  o,
		cache: make(map[string]cacheEntry),
		stop:  make(chan struct{}),
	}
	if o.cleanupInterval > 0 {
		c.wg.Go(func() {
			c.cleanupLoop(o.cleanupInterval)
		})
	}
	return c
}

// Stop stops the background cleanup goroutine.
func (c *CachingReadFileFS) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	close(c.stop)
	c.wg.Wait(ctx)
	return nil
}

func (c *CachingReadFileFS) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var expired []string

			c.mu.RLock()
			for k, v := range c.cache {
				if now.After(v.expires) {
					expired = append(expired, k)
				}
			}
			c.mu.RUnlock()

			if len(expired) > 0 {
				c.mu.Lock()
				for _, k := range expired {
					// Double-check expiration under write lock before deleting,
					// in case the entry was refreshed while we were checking.
					if v, ok := c.cache[k]; ok && now.After(v.expires) {
						delete(c.cache, k)
					}
				}
				c.mu.Unlock()
			}
		case <-c.stop:
			return
		}
	}
}

// ReadFile reads the named file, utilizing the cache if the entry is fresh.
func (c *CachingReadFileFS) ReadFile(name string) ([]byte, error) {
	return c.ReadFileCtx(context.Background(), name)
}

func (c *CachingReadFileFS) readFileAndUpdateCache(ctx context.Context, name string) ([]byte, error) {
	data, err := c.fs.ReadFileCtx(ctx, name)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[name] = cacheEntry{
		data:    data,
		expires: time.Now().Add(c.opts.ttl),
	}
	return data, nil

}

// ReadFileCtx reads the named file using the provided context, utilizing the cache if fresh.
func (c *CachingReadFileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	c.mu.RLock()
	entry, ok := c.cache[name]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expires) {
		return bytes.Clone(entry.data), nil
	}

	if !c.opts.singleFlight {
		data, err := c.readFileAndUpdateCache(ctx, name)
		if err != nil {
			return nil, err
		}
		return bytes.Clone(data), nil
	}

	data, err, _ := c.sf.Do(ctx, name, func() (any, error) {
		return c.readFileAndUpdateCache(ctx, name)
	})
	if err != nil {
		return nil, err
	}

	return bytes.Clone(data.([]byte)), nil
}

// Forget removes the named file from the cache.
func (c *CachingReadFileFS) Forget(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, name)
}

// SingleFlightReadFileFS is a wrapper around a ReadFileFS that provides single-flight
// behavior for concurrent calls to ReadFileCtx and ReadFile with the same name. This
// can be used in conjunction with CachingReadFileFS to prevent thundering herd issues
// on cache misses.
type SingleFlightReadFileFS struct {
	fs file.ReadFileFS
	sf ctxsync.SingleFlight
}

func NewSingleFlightReadFileFS(fs file.ReadFileFS) *SingleFlightReadFileFS {
	return &SingleFlightReadFileFS{fs: fs}
}

func (s *SingleFlightReadFileFS) ReadFile(name string) ([]byte, error) {
	return s.ReadFileCtx(context.Background(), name)
}

func (s *SingleFlightReadFileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	v, err, _ := s.sf.Do(ctx, name, func() (any, error) {
		return s.fs.ReadFileCtx(ctx, name)
	})
	if err != nil {
		return nil, err
	}
	return bytes.Clone(v.([]byte)), nil
}
