// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxsync

import (
	"context"
	"errors"

	"golang.org/x/sync/singleflight"
)

// SingleFlight mirrors golang.org/x/sync/singleflight.Group but with different handling of
// context cancelation. In particular, if a shared invocation returns with
// with a canceled context, but the caller's context is not canceled, the
// group will reissue the invocation.
type SingleFlight struct {
	g *singleflight.Group
}

// New creates a new Group with the provided backoff strategy. The backoff will be used
// to determine whether to retry an operation that failed with a retryable error, and
// how long to wait before retrying the operation. If the backoff returns true, no more
// retries will be attempted.
func New() *SingleFlight {
	return &SingleFlight{
		g: &singleflight.Group{},
	}
}

// Do is like singleflight.Group.Do but with different handling of context
// cancellation. In particular, if a shared invocation returns with a canceled context,
// but the caller's context is not canceled, the group will reissue the invocation.
func (g *SingleFlight) Do(ctx context.Context, key string, fn func() (any, error)) (v any, err error, shared bool) {
	v, err, shared = g.g.Do(key, func() (any, error) {
		return fn()
	})
	if err == nil || !shared {
		// there was no error or the error was not shared.
		return v, err, shared
	}
	if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() == nil {
		return g.g.Do(key, fn)
	}
	return v, err, shared
}

// DocChan is like singleflight.Group.DoChan but with different handling of context
// cancellation. In particular, if a shared invocation returns with a canceled context,
// but the caller's context is not canceled, the group will reissue the invocation.
func (g *SingleFlight) DoChan(ctx context.Context, key string, fn func() (any, error)) <-chan singleflight.Result {
	ch := g.g.DoChan(key, func() (any, error) {
		return fn()
	})
	select {
	case res := <-ch:
		if (errors.Is(res.Err, context.Canceled) || errors.Is(res.Err, context.DeadlineExceeded)) && ctx.Err() == nil {
			return g.g.DoChan(key, fn)
		}
		nch := make(chan singleflight.Result, 1)
		nch <- res
		return nch
	case <-ctx.Done():
		return ch
	}
}

func (g *SingleFlight) Forget(key string) {
	g.g.Forget(key)
}
