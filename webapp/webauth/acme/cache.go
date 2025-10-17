// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package acme

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cloudeng.io/errors"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/os/lockedfile"
	"golang.org/x/crypto/acme/autocert"
)

// Cache implements autocert.Cache with file locking to allow
// safe concurrent access to the underlying cache in order to
// extract certificates programmatically.
type Cache struct {
	lock         *lockedfile.Mutex
	localCache   autocert.Cache
	backingStore CacheFS
	readonly     bool
}

// ErrCacheMiss is the same as autocert.ErrCacheMiss
var ErrCacheMiss = autocert.ErrCacheMiss

// CacheFS defines an interface that combines reading, writing
// and deleting files and is used to create an acme/autocert cache.
type CacheFS interface {
	ReadFileCtx(ctx context.Context, name string) ([]byte, error)
	WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
	Delete(ctx context.Context, name string) error
}

// NewCache returns an instance of autocert.Cache that will store
// certificates in 'backing' store, but use the local file system for
// temporary/private data such as the ACME client's private key. This
// allows for certificates to be shared across multiple hosts by using
// a distributed 'backing' store such as AWS' secretsmanager.
// Certificates may be extracted safely for use by other servers
// by using the readonly option.
func NewCache(localDir string, storeFS CacheFS, readonly bool) *Cache {
	if !readonly {
		if err := os.MkdirAll(localDir, 0700); err != nil {
			panic(err)
		}
	}
	return &Cache{
		lock:         lockedfile.MutexAt(filepath.Join(localDir, "dir.lock")),
		localCache:   autocert.DirCache(localDir),
		backingStore: storeFS,
		readonly:     readonly,
	}
}

// IsLocalName returns true if the specified name is for local-only
// data such as ACME client private keys or http-01 challenge tokens.
func IsLocalName(name string) bool {
	return strings.HasSuffix(name, "+token") ||
		strings.HasSuffix(name, "+rsa") ||
		strings.Contains(name, "http-01") ||
		(strings.HasPrefix(name, "acme_account") &&
			strings.HasSuffix(name, "key"))
}

var (
	ErrReadonlyCache    = errors.New("readonly cache")
	ErrLocalOperation   = errors.New("local operation")
	ErrBackingOperation = errors.New("backing store operation")
	ErrLockFailed       = errors.New("lock acquisition failed")
)

// Delete implements autocert.Cache.
func (dc *Cache) Delete(ctx context.Context, name string) error {
	if dc.readonly {
		return fmt.Errorf("delete %q: %w", name, ErrReadonlyCache)
	}
	if !IsLocalName(name) {
		if err := dc.backingStore.Delete(ctx, name); err != nil {
			return fmt.Errorf("delete %q: %w", name, errors.NewM(err, ErrBackingOperation))
		}
		return nil
	}
	unlock, err := dc.lock.Lock()
	if err != nil {
		return errors.NewM(fmt.Errorf("lock acquisition failed: %w", err), ErrLockFailed)
	}
	defer unlock()
	if err := dc.localCache.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete %q: %w", name, errors.NewM(err, ErrLocalOperation))
	}
	return nil

}

// Get implements autocert.Cache.
func (dc *Cache) Get(ctx context.Context, name string) ([]byte, error) {
	if !IsLocalName(name) {
		data, err := dc.backingStore.ReadFileCtx(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("get %q: %w", name, errors.NewM(err, ErrBackingOperation))
		}
		return data, nil
	}
	var err error
	var unlock func()
	if dc.readonly {
		unlock, err = dc.lock.RLock()
	} else {
		unlock, err = dc.lock.Lock()
	}
	if err != nil {
		return nil, errors.NewM(fmt.Errorf("lock acquisition failed: %w", err), ErrLockFailed)
	}
	defer unlock()
	data, err := dc.localCache.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get %q: %w", name, errors.NewM(err, ErrLocalOperation))
	}
	return data, nil
}

// Put implements autocert.Cache.
func (dc *Cache) Put(ctx context.Context, name string, data []byte) error {
	if dc.readonly {
		return fmt.Errorf("put %q: %w", name, ErrReadonlyCache)
	}
	if !IsLocalName(name) {
		if err := dc.backingStore.WriteFileCtx(ctx, name, data, 0600); err != nil {
			return fmt.Errorf("put %q: %w", name, errors.NewM(err, ErrBackingOperation))
		}
		return nil
	}
	unlock, err := dc.lock.Lock()
	if err != nil {
		return errors.NewM(fmt.Errorf("lock acquisition failed: %w", err), ErrLockFailed)
	}
	defer unlock()
	if err := dc.localCache.Put(ctx, name, data); err != nil {
		ctxlog.Logger(ctx).Error("acme.Cache.Put failed", "key", name, "error", err)
		return fmt.Errorf("put %q: %w", name, errors.NewM(err, ErrLocalOperation))
	}
	ctxlog.Logger(ctx).Error("acme.Cache.Put succeeded", "key", name)
	return nil
}

// NewNullCache returns an autocert.Cache that never stores any data and is
// intended for use when testing.
func NewNullCache() autocert.Cache {
	return &nullcache{}
}

type nullcache struct{}

// Delete implements autocert.Cache.
func (nc *nullcache) Delete(_ context.Context, _ string) error {
	return nil
}

// Get implements autocert.Cache.
func (nc *nullcache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

// Put implements autocert.Cache.
func (nc *nullcache) Put(_ context.Context, _ string, _ []byte) error {
	return nil
}
