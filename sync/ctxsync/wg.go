// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ctxsync provides context aware synchronisation primitives.
package ctxsync

import (
	"context"
	"sync"
)

// WaitGroup represents a context aware sync.WaitGroup
type WaitGroup struct {
	sync.WaitGroup
}

// Wait blocks until the WaitGroup reaches zero or the
// context is canceled, whichever comes first.
func (wg *WaitGroup) Wait(ctx context.Context) {
	ch := make(chan struct{})
	go func() {
		wg.WaitGroup.Wait()
		close(ch)
	}()
	select {
	case <-ch:
	case <-ctx.Done():
	}
}
