// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package registry provides support for various forms of registry
// useful for building command line tools.
package registry

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// New is a function that creates a new instance of type T
type New[T any] func(ctx context.Context, args ...any) (T, error)

type item[T any] struct {
	key     string
	factory New[T]
}

// T represents a registry for a specific type T that
// selected using a string key, which is typically a URI scheme.
type T[T any] struct {
	mu    sync.RWMutex
	items []item[T]
}

// Register registers a new factory function for the given key.
func (r *T[T]) Register(key string, factory New[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, item[T]{key: key, factory: factory})
}

// Get retrieves the factory function for the given key, or
// nil if the key is not registered.
func (r *T[T]) Get(key string) New[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if item.key == key {
			return item.factory
		}
	}
	return nil
}

// ErrUnknownKey is returned when an unregistered key
// is encountered.
var ErrUnknownKey = errors.New("unregistered key")

// ConvertAnyArgs converts a variadic list of any to a slice
// of the specified type T.
func ConvertAnyArgs[T any](args ...any) []T {
	result := make([]T, 0, len(args))
	for _, arg := range args {
		if v, ok := arg.(T); ok {
			result = append(result, v)
		}
	}
	return result
}

// Scheme extracts the scheme from the given path, returning
// "file" if no scheme is present.
func Scheme(path string) string {
	for i, c := range path {
		if c == ':' {
			return strings.ToLower(path[:i])
		}
		if c == '/' || c == '\\' {
			break
		}
	}
	return "file"
}
