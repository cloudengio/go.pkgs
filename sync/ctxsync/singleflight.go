// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxsync

import (
	"context"
	"errors"

	"golang.org/x/sync/singleflight"
)

// SingleFlight mirrors golang.org/x/sync/singleflight.Group but with different
// handling of context cancellation. In particular, if a shared invocation returns
// with a canceled or timed out context, but the caller's context is not canceled,
// SingleFlight will reissue the invocation. This handles the case where one
// invocation has its context canceled, but others have not and hence could
// potentially succeed if reissued.
type SingleFlight struct {
	g singleflight.Group
}

// New creates a new SingleFlight instance.
func New() *SingleFlight {
	return &SingleFlight{}
}

// Do is like singleflight.Group.Do but with different handling of context
// cancellation. In particular, if a shared invocation returns with a canceled
// or timed out context, but the caller's context is not canceled,
// SingleFlight will reissue the invocation.
func (g *SingleFlight) Do(ctx context.Context, key string, fn func() (any, error)) (v any, err error, shared bool) { //nolint:revive // ignore error should be last to mirror singleflight.Group.Do signature
	for {
		v, err, shared = g.g.Do(key, fn)
		if shared && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() == nil {
			continue
		}
		return
	}
}

// DoChan is like singleflight.Group.DoChan but with different handling of context
// cancellation. In particular, if a shared invocation returns with a canceled
// or timed out context, but the caller's context is not canceled,
// SingleFlight will reissue the invocation.
func (g *SingleFlight) DoChan(ctx context.Context, key string, fn func() (any, error)) <-chan singleflight.Result {
	out := make(chan singleflight.Result, 1)
	go func() {
		for {
			ch := g.g.DoChan(key, fn)
			select {
			case res := <-ch:
				if res.Shared && (errors.Is(res.Err, context.Canceled) || errors.Is(res.Err, context.DeadlineExceeded)) && ctx.Err() == nil {
					// The invocation was shared and the error was a cancellation error, but our context is not canceled. Retry the invocation.
					continue
				}
				out <- res
				return
			case <-ctx.Done():
				out <- singleflight.Result{Err: ctx.Err()}
				return
			}
		}
	}()
	return out
}

func (g *SingleFlight) Forget(key string) {
	g.g.Forget(key)
}
