// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import "context"

type contextKey[T any] struct{}

// ContextWithValue returns a context carrying v, which is retrieved by
// ContextValue. The value is keyed by its type, so a type that is used to
// carry a value should be specific to it: a named type rather than a string,
// for example.
func ContextWithValue[T any](ctx context.Context, v T) context.Context {
	return context.WithValue(ctx, contextKey[T]{}, v)
}

// ContextValue returns the value of type T carried by ctx, if any.
func ContextValue[T any](ctx context.Context) (T, bool) {
	if ctx == nil {
		var zero T
		return zero, false
	}
	v, ok := ctx.Value(contextKey[T]{}).(T)
	return v, ok
}
