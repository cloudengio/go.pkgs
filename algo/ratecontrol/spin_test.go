// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
)

// fakeClock is a controllable clock for deterministic tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{now: time.Now()} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestSpinDetector(t *testing.T) {
	const (
		maxIterations = int64(3)
		period        = 100 * time.Millisecond
	)
	clk := newFakeClock()
	sd := ratecontrol.NewSpinDetector(maxIterations, period, ratecontrol.WithClock(clk.Now))

	// Calls below the threshold must not be reported as spinning.
	for i := range maxIterations - 1 {
		if sd.Tick() {
			t.Errorf("tick %d: expected no spin below threshold", i)
		}
	}

	// The max-th call within the period must be detected as a spin.
	if !sd.Tick() {
		t.Error("expected spin at threshold")
	}

	// After the reset triggered by spin detection the count is back to zero;
	// the next (max-1) calls must not trigger again.
	for i := range maxIterations - 1 {
		if sd.Tick() {
			t.Errorf("post-spin tick %d: expected no spin after reset", i)
		}
	}

	// A second spin within the same (reset) period must also be detected.
	if !sd.Tick() {
		t.Error("expected second spin")
	}
}

func TestSpinDetectorPeriodExpiry(t *testing.T) {
	const (
		maxIterations = int64(3)
		period        = 100 * time.Millisecond
	)
	clk := newFakeClock()
	sd := ratecontrol.NewSpinDetector(maxIterations, period, ratecontrol.WithClock(clk.Now))

	// Make (max-1) calls so the count is one shy of the threshold.
	for range maxIterations - 1 {
		sd.Tick()
	}

	// Advance past the period: the next call reaches the threshold but the
	// window has expired, so it should NOT count as a spin and must reset state.
	clk.Advance(period + time.Millisecond)
	if sd.Tick() {
		t.Error("expected no spin when the period has already expired")
	}

	// The reset puts us into a fresh window. A full burst should now be detectable.
	for range maxIterations - 1 {
		if sd.Tick() {
			t.Error("expected no spin in the new period (below threshold)")
		}
	}
	if !sd.Tick() {
		t.Error("expected spin in the new period")
	}
}

func TestSpinDetectorConcurrent(t *testing.T) {
	// Run under -race to verify there are no data races.
	const (
		maxIterations = int64(10)
		period        = 50 * time.Millisecond
		workers       = 20
		ticks         = 100
	)
	clk := newFakeClock()
	sd := ratecontrol.NewSpinDetector(maxIterations, period, ratecontrol.WithClock(clk.Now))

	var detected atomic.Int64
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range ticks {
				if sd.Tick() {
					detected.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if detected.Load() == 0 {
		t.Error("expected at least one spin to be detected under concurrent load")
	}
}

// countingBackoff is a Backoff that counts how many times Wait is called.
type countingBackoff struct {
	calls   int
	retries int
}

func (cb *countingBackoff) Wait(_ context.Context, _ any) (bool, error) {
	cb.calls++
	cb.retries++
	return false, nil
}

func (cb *countingBackoff) Retries() int { return cb.retries }

func TestBackoffOnSpin(t *testing.T) {
	const (
		maxIterations = int64(3)
		period        = 100 * time.Millisecond
	)
	clk := newFakeClock()
	cb := &countingBackoff{}
	bos := ratecontrol.NewBackoffOnSpin(maxIterations, period, cb, ratecontrol.WithClock(clk.Now))

	ctx := context.Background()

	// Calls below max must not trigger the backoff.
	for i := range maxIterations - 1 {
		stopped, err := bos.Wait(ctx, nil)
		if err != nil || stopped {
			t.Fatalf("below-threshold wait %d: got stopped=%v err=%v", i, stopped, err)
		}
	}
	if cb.calls != 0 {
		t.Errorf("expected no backoff calls below threshold, got %d", cb.calls)
	}

	// The max-th call must trigger the backoff.
	stopped, err := bos.Wait(ctx, nil)
	if err != nil || stopped {
		t.Fatalf("spin wait: got stopped=%v err=%v", stopped, err)
	}
	if cb.calls != 1 {
		t.Errorf("expected 1 backoff call on spin, got %d", cb.calls)
	}

	// Retries delegates to the underlying Backoff.
	if got, want := bos.Retries(), cb.Retries(); got != want {
		t.Errorf("Retries(): got %v, want %v", got, want)
	}
}

func TestBackoffOnSpinContextCancel(t *testing.T) {
	const (
		maxIterations = int64(1)
		period        = 100 * time.Millisecond
	)
	clk := newFakeClock()
	// Use a real backoff that will block until the context is cancelled.
	backoff := ratecontrol.NewExponentialBackoff(time.Hour, 10)
	bos := ratecontrol.NewBackoffOnSpin(maxIterations, period, backoff, ratecontrol.WithClock(clk.Now))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the blocking Wait returns right away

	stopped, err := bos.Wait(ctx, nil)
	if err == nil {
		t.Error("expected context-cancel error, got nil")
	}
	if !stopped {
		t.Error("expected stopped=true when context is cancelled")
	}
}
