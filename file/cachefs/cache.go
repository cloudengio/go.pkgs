// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cachefs provides a caching layer for ReadFileFS implementations.
package cachefs

import (
	"context"
	"sync"
	"time"

	"cloudeng.io/file"
)

type cacheEntry struct {
	data    []byte
	expires time.Time
}

// CachingReadfileFS implements a caching layer over a ReadFileFS.
type CachingReadfileFS struct {
	fs   file.ReadFileFS
	ttl  time.Duration
	stop chan struct{}

	mu    sync.RWMutex
	cache map[string]cacheEntry
	wg    sync.WaitGroup
}

type options struct {
	ttl             time.Duration
	cleanupInterval time.Duration
}
type Option func(*options)

const (
	DefaultTTL             = 24 * time.Hour
	DefaultCleanupInterval = 1 * time.Hour
)

// WithTTL specifies the time-to-live for cache entries. The default is 5 minutes.
func WithTTL(d time.Duration) Option {
	return func(o *options) {
		o.ttl = d
	}
}

// WithCleanupInterval specifies the interval for periodic cleanup of expired cache entries. The default is 1 minute.
func WithCleanupInterval(d time.Duration) Option {
	return func(o *options) {
		o.cleanupInterval = d
	}
}

// NewCachingReadFileFS creates a new CachingReadfileFS with the specified TTL
// and cleanup interval. It starts a background goroutine to periodically clear
// out expired cache entries. Call Close to stop the background goroutine.
func NewCachingReadFileFS(fs file.ReadFileFS, opts ...Option) *CachingReadfileFS {
	o := &options{
		ttl:             DefaultTTL,
		cleanupInterval: DefaultCleanupInterval,
	}
	for _, fn := range opts {
		fn(o)
	}
	c := &CachingReadfileFS{
		fs:    fs,
		ttl:   o.ttl,
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

// Close stops the background cleanup goroutine.
func (c *CachingReadfileFS) Close() error {
	close(c.stop)
	c.wg.Wait()
	return nil
}

func (c *CachingReadfileFS) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.cache {
				if now.After(v.expires) {
					delete(c.cache, k)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}

// ReadFile reads the named file, utilizing the cache if the entry is fresh.
func (c *CachingReadfileFS) ReadFile(name string) ([]byte, error) {
	return c.ReadFileCtx(context.Background(), name)
}

// ReadFileCtx reads the named file using the provided context, utilizing the cache if fresh.
func (c *CachingReadfileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	c.mu.RLock()
	entry, ok := c.cache[name]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expires) {
		return entry.data, nil
	}

	data, err := c.fs.ReadFileCtx(ctx, name)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[name] = cacheEntry{
		data:    data,
		expires: time.Now().Add(c.ttl),
	}

	return data, nil
}

// Invalidate removes the named file from the cache.
func (c *CachingReadfileFS) Invalidate(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, name)
}
