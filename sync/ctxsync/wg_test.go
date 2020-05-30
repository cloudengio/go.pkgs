// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxsync_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloudeng.io/sync/ctxsync"
)

func ExampleWaitGrouo() {
	var wg ctxsync.WaitGroup
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	wg.Wait(ctx)
	fmt.Println(ctx.Err())
	// Output:
	// context canceled
}

func TestWaitGroupInline(t *testing.T) {
	var wg ctxsync.WaitGroup
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wg.Wait(ctx)
	if got, want := ctx.Err().Error(), "context canceled"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWaitGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg ctxsync.WaitGroup
	wg.Add(1)
	var out string
	go func() {
		out = "done"
		wg.Done()
	}()
	wg.Wait(ctx)
	if got, want := out, "done"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	cancel()
	wg.Add(1)
	wg.Wait(ctx)
	// The test will timeout if we never get here.
}
