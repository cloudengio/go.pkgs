// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloudeng.io/os/executil"
)

const tick = time.Millisecond

func TestWaitFor_InvalidInterval(t *testing.T) {
	check := func(_ context.Context) (bool, error) { return true, nil }
	for _, interval := range []time.Duration{0, -1, -time.Second} {
		if err := executil.WaitFor(context.Background(), interval, check); err == nil {
			t.Errorf("interval %v: expected error, got nil", interval)
		}
	}
}

func TestWaitFor_ImmediateDone(t *testing.T) {
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return true, nil
	}
	if err := executil.WaitFor(context.Background(), tick, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want 1", calls)
	}
}

func TestWaitFor_ImmediateDoneWithError(t *testing.T) {
	want := errors.New("done but failed")
	check := func(_ context.Context) (bool, error) { return true, want }
	err := executil.WaitFor(context.Background(), tick, check)
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestWaitFor_PollsUntilDone(t *testing.T) {
	const target = 3
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return calls >= target, nil
	}
	if err := executil.WaitFor(context.Background(), tick, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != target {
		t.Errorf("check called %d times, want %d", calls, target)
	}
}

func TestWaitFor_ContextCancelledWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		if calls == 2 {
			cancel()
		}
		return false, nil
	}

	err := executil.WaitFor(ctx, tick, check)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWaitFor_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The initial check fires even when the context is already done.
	// If that check returns done=false, the subsequent select immediately
	// picks ctx.Done() and returns context.Canceled.
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return false, nil
	}

	err := executil.WaitFor(ctx, tick, check)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want exactly 1 (the pre-select call)", calls)
	}
}

func TestWaitFor_CheckReceivesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var gotCtx context.Context
	check := func(c context.Context) (bool, error) {
		gotCtx = c
		return true, nil
	}
	if err := executil.WaitFor(ctx, tick, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCtx != ctx {
		t.Error("check did not receive the original context")
	}
}

func TestWaitFor_TransientErrorContinues(t *testing.T) {
	// When check returns (false, non-nil err), WaitFor keeps polling
	// rather than returning immediately — the done flag controls termination.
	transient := errors.New("transient")
	const target = 4
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		if calls < target {
			return false, transient // not done, but has an error
		}
		return true, nil
	}
	if err := executil.WaitFor(context.Background(), tick, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != target {
		t.Errorf("check called %d times, want %d", calls, target)
	}
}

func TestWaitFor_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*tick)
	defer cancel()

	err := executil.WaitFor(ctx, tick, func(_ context.Context) (bool, error) {
		return false, nil // never done
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
}
