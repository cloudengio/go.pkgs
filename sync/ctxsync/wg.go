// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ctxsync provides context aware synchronisation primitives.
package ctxsync

import (
	"context"
	"sync"
)

// WaitGroup is a context-aware sync.WaitGroup. The zero value is ready to use.
// Unlike sync.WaitGroup, it is safe to call Add immediately after a Wait that
// returned due to context cancellation.
type WaitGroup struct {
	mu    sync.Mutex
	count int
	done  chan struct{}
}

// Add adds delta to the WaitGroup counter. If the counter transitions from
// zero to positive a new completion channel is allocated. If the counter
// transitions from positive to zero all blocked Wait calls are unblocked.
// It panics if the counter goes negative.
func (wg *WaitGroup) Add(delta int) {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	prev := wg.count
	wg.count += delta
	if wg.count < 0 {
		panic("sync: negative WaitGroup counter")
	}
	switch {
	case prev == 0 && wg.count > 0:
		wg.done = make(chan struct{})
	case wg.count == 0 && prev > 0:
		close(wg.done)
	}
}

// Done decrements the WaitGroup counter by one.
func (wg *WaitGroup) Done() { wg.Add(-1) }

// Go calls f in a new goroutine and adds that task to the WaitGroup. When f
// returns, the task is removed from the WaitGroup. If f panics, the task is
// not removed to ensure the panic remains fatal.
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				// f panicked. Calling Done would unblock Wait and allow the
				// main goroutine to exit before the panic propagates, so
				// re-panic without calling Done to keep the crash fatal.
				panic(x)
			}
			wg.Done()
		}()
		f()
	}()
}

// Wait blocks until the WaitGroup counter reaches zero or the context is
// canceled, whichever comes first.
func (wg *WaitGroup) Wait(ctx context.Context) {
	wg.mu.Lock()
	if wg.count == 0 {
		wg.mu.Unlock()
		return
	}
	ch := wg.done
	wg.mu.Unlock()
	select {
	case <-ch:
	case <-ctx.Done():
	}
}
