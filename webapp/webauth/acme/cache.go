// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package acme

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloudeng.io/os/lockedfile"
	"cloudeng.io/webapp"
	"golang.org/x/crypto/acme/autocert"
)

type dircache struct {
	lock     *lockedfile.Mutex
	cache    autocert.Cache
	readonly bool
}

// ErrCacheMiss is the same as autocert.ErrCacheMiss
var ErrCacheMiss = autocert.ErrCacheMiss

// NewDirCache returns an instance of a local filesystem based
// cache for certificates and the acme account key but with
// file system locking. Set the readonly argument for readonly
// access via the 'Get' method, this will typically be used to
// safely extract keys for use by other servers. However, ideally,
// a secure shared services such as Amazon's secrets manager should
// be used instead.
func NewDirCache(dir string, readonly bool) autocert.Cache {
	if !readonly {
		if err := os.MkdirAll(dir, 0700); err != nil {
			panic(err)
		}
	}
	return &dircache{
		lock:     lockedfile.MutexAt(filepath.Join(dir, "dir.lock")),
		cache:    autocert.DirCache(dir),
		readonly: readonly,
	}
}

// Delete implements autocert.Cache.
func (dc *dircache) Delete(ctx context.Context, name string) error {
	if dc.readonly {
		return fmt.Errorf("readonly cache")
	}
	unlock, err := dc.lock.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	return dc.cache.Delete(ctx, name)
}

// Get implements autocert.Cache.
func (dc *dircache) Get(ctx context.Context, name string) ([]byte, error) {
	var err error
	var unlock func()
	if dc.readonly {
		unlock, err = dc.lock.RLock()
	} else {
		unlock, err = dc.lock.Lock()
	}
	if err != nil {
		return nil, err
	}
	defer unlock()
	return dc.cache.Get(ctx, name)
}

// Put implements autocert.Cache.
func (dc *dircache) Put(ctx context.Context, name string, data []byte) error {
	if dc.readonly {
		return fmt.Errorf("readonly cache")
	}
	unlock, err := dc.lock.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	return dc.cache.Put(ctx, name, data)
}

// NewNullCache returns an autocert.Cache that never stores any data and is
// intended for use when testing.
func NewNullCache() autocert.Cache {
	return &nullcache{}
}

type nullcache struct{}

// Delete implements autocert.Cache.
func (nc *nullcache) Delete(ctx context.Context, name string) error {
	return nil
}

// Get implements autocert.Cache.
func (nc *nullcache) Get(ctx context.Context, name string) ([]byte, error) {
	return nil, ErrCacheMiss
}

// Put implements autocert.Cache.
func (nc *nullcache) Put(ctx context.Context, name string, data []byte) error {
	return nil
}

const (
	dirCacheName  = "autocert-dir-cache"
	nullCacheName = "autocert-null-cache"
)

var (
	// AutoCertDiskStore creates instances of webapp.CertStore using
	// NewDirCache with read-only set to true.
	AutoCertDiskStore = CertStoreFactory{dirCacheName}

	// AutoCertNullStore creates instances of webapp.CertStore using
	// NewNullCache.
	AutoCertNullStore = CertStoreFactory{nullCacheName}
)

// CertStoreFactory represents the webapp.CertStore's that can be
// created by this package.
type CertStoreFactory struct {
	typ string
}

// Type implements webapp.CertStoreFactory.
func (f CertStoreFactory) Type() string {
	return f.typ
}

func unsupported(typ string) string {
	return fmt.Sprintf(
		"unsupported factory type: %s: use one of %s", typ, strings.Join([]string{dirCacheName, nullCacheName}, ","))
}

// New implements webapp.CertStoreFactory.
func (f CertStoreFactory) New(ctx context.Context, dir string, opts ...interface{}) (webapp.CertStore, error) {
	switch f.typ {
	case dirCacheName:
		return NewDirCache(dir, true), nil
	case nullCacheName:
		return NewNullCache(), nil
	}
	return nil, errors.New(unsupported(f.typ))
}

// Describe implements webapp.CertStoreFactory.
func (f CertStoreFactory) Describe() string {
	switch f.typ {
	case dirCacheName:
		return dirCacheName + " retrieves certificates from a local filesystem instance of an acme/autocert cache"
	case nullCacheName:
		return nullCacheName + " never stores any certificates and always returns a cache miss, use it for testing"
	}
	panic(unsupported(f.typ))
}
