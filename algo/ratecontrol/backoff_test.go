// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
)

func TestBackoffOffset(t *testing.T) {
	ctx := context.Background()
	numRetries := 10
	bo := ratecontrol.NewExponentialBackoffOffset(time.Millisecond, numRetries)

	for i := range numRetries {
		done, err := bo.Wait(ctx, nil)
		if err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		if done {
			t.Fatalf("expected to not be done on retry %d", i)
		}
	}

	done, err := bo.Wait(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("expected to be done after max steps")
	}

	if got, want := bo.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if !bo.Done() {
		t.Error("expected Done after Wait has returned true")
	}
}

// nextLoop runs the select loop documented on Backoff.Next and returns the
// number of retries granted before Done became true.
func nextLoop(t *testing.T, b ratecontrol.Backoff, maxRetries int) int {
	t.Helper()
	retries := 0
	for {
		<-b.Next()
		if b.Done() {
			return retries
		}
		retries++
		if retries > maxRetries {
			t.Fatalf("too many retries: %v", retries)
		}
	}
}

func TestBackoffNext(t *testing.T) {
	numRetries := 3
	initial := time.Millisecond
	eb := ratecontrol.NewExponentialBackoff(initial, numRetries)

	if eb.Done() {
		t.Error("expected Done to be false before any retries")
	}

	start := time.Now()
	retries := nextLoop(t, eb, numRetries)

	// The full delay budget must be consumed: initial + 2*initial + 4*initial.
	if elapsed, minElapsed := time.Since(start), 7*initial; elapsed < minElapsed {
		t.Errorf("elapsed %v, expected at least %v", elapsed, minElapsed)
	}
	if got, want := retries, numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := eb.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Once done, Next must remain immediately ready and Done must stay true.
	select {
	case <-eb.Next():
	case <-time.After(time.Second):
		t.Error("Next did not fire immediately after Done")
	}
	if !eb.Done() {
		t.Error("expected Done to remain true")
	}
}

func TestBackoffNextOffset(t *testing.T) {
	numRetries := 4
	bo := ratecontrol.NewExponentialBackoffOffset(time.Millisecond, numRetries)

	retries := nextLoop(t, bo, numRetries)

	if got, want := retries, numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := bo.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if !bo.Done() {
		t.Error("expected Done after all retries are exhausted")
	}
}

// TestBackoffRandomizedOffsetOption covers both values of
// WithRandomizedOffset: true draws the first delay from (0, initial) instead
// of using initial itself, and false leaves it at initial, including when it
// overrides the offset that NewExponentialBackoffOffset applies.
func TestBackoffRandomizedOffsetOption(t *testing.T) {
	ctx := context.Background()
	initial := 20 * time.Millisecond

	// The first delay is drawn from (0, initial), so across several trials at
	// least one must come in under initial. The chance of every trial landing
	// in the upper half is 2^-16.
	shorter := false
	for range 16 {
		bo := ratecontrol.NewExponentialBackoff(initial, 3, ratecontrol.WithRandomizedOffset(true))
		start := time.Now()
		done, err := bo.Wait(ctx, nil)
		if err != nil || done {
			t.Fatalf("Wait: done=%v, err=%v", done, err)
		}
		if time.Since(start) < initial {
			shorter = true
			break
		}
	}
	if !shorter {
		t.Errorf("the first delay was never shorter than %v, want it randomized", initial)
	}

	// Whenever the offset is not enabled the first delay is the full initial
	// interval, whether the option is absent, explicitly false, or false
	// overriding the offset applied by NewExponentialBackoffOffset.
	for _, tc := range []struct {
		name string
		bo   ratecontrol.Backoff
	}{
		{"absent", ratecontrol.NewExponentialBackoff(initial, 3)},
		{"false", ratecontrol.NewExponentialBackoff(initial, 3, ratecontrol.WithRandomizedOffset(false))},
		{"false overriding the offset constructor",
			ratecontrol.NewExponentialBackoffOffset(initial, 3, ratecontrol.WithRandomizedOffset(false))},
	} {
		start := time.Now()
		done, err := tc.bo.Wait(ctx, nil)
		if err != nil || done {
			t.Fatalf("%v: Wait: done=%v, err=%v", tc.name, done, err)
		}
		if elapsed := time.Since(start); elapsed < initial {
			t.Errorf("%v: first delay %v, want at least %v", tc.name, elapsed, initial)
		}
	}
}

// TestBackoffUnlimitedRetries covers WithUnlimitedRetries: the backoff is never
// done, however many retries it records.
func TestBackoffUnlimitedRetries(t *testing.T) {
	ctx := context.Background()
	steps := 3
	eb := ratecontrol.NewExponentialBackoff(time.Millisecond, steps,
		ratecontrol.WithUnlimitedRetries(true))

	retries := steps * 3
	for i := range retries {
		done, err := eb.Wait(ctx, nil)
		if err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		if done {
			t.Fatalf("retry %d: Wait returned true, want an unlimited backoff to continue", i)
		}
		if eb.Done() {
			t.Fatalf("retry %d: Done returned true, want an unlimited backoff to continue", i)
		}
	}
	if got, want := eb.Retries(), retries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestBackoffUnlimitedRetriesMaxDelay verifies that an unlimited backoff pins
// the delay at its maximum, initial * 2^(steps-1), once steps is reached rather
// than doubling it indefinitely.
func TestBackoffUnlimitedRetriesMaxDelay(t *testing.T) {
	ctx := context.Background()
	initial := 10 * time.Millisecond
	steps := 3
	maxDelay := initial * (1 << (steps - 1))
	eb := ratecontrol.NewExponentialBackoff(initial, steps,
		ratecontrol.WithUnlimitedRetries(true))

	// Consume the growing phase: initial, 2*initial, 4*initial.
	for i := range steps {
		if done, err := eb.Wait(ctx, nil); err != nil || done {
			t.Fatalf("retry %d: done=%v, err=%v", i, done, err)
		}
	}

	const extra = 3
	start := time.Now()
	for i := range extra {
		if done, err := eb.Wait(ctx, nil); err != nil || done {
			t.Fatalf("extra retry %d: done=%v, err=%v", i, done, err)
		}
	}
	elapsed := time.Since(start)
	if minElapsed := extra * maxDelay; elapsed < minElapsed {
		t.Errorf("elapsed %v, want at least %v (%v per retry)", elapsed, minElapsed, maxDelay)
	}
	// Doubling past the cap would spend 2+4+8 = 14 maxDelays here, not 3.
	if maxElapsed := 3 * extra * maxDelay; elapsed > maxElapsed {
		t.Errorf("elapsed %v, want less than %v: the delay must stay pinned at %v", elapsed, maxElapsed, maxDelay)
	}
}

// TestBackoffUnlimitedRetriesNext verifies that Next never reports completion
// for an unlimited backoff.
func TestBackoffUnlimitedRetriesNext(t *testing.T) {
	eb := ratecontrol.NewExponentialBackoff(time.Millisecond, 2,
		ratecontrol.WithUnlimitedRetries(true))

	retries := 6
	for i := range retries {
		if _, ok := <-eb.Next(); !ok {
			t.Fatalf("retry %d: Next returned a closed channel, want an unlimited backoff to continue", i)
		}
		if eb.Done() {
			t.Fatalf("retry %d: Done returned true, want an unlimited backoff to continue", i)
		}
	}
	if got, want := eb.Retries(), retries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestBackoffOffsetOptions verifies that NewExponentialBackoffOffset applies the
// options it is given in addition to the randomized offset it implies.
func TestBackoffOffsetOptions(t *testing.T) {
	ctx := context.Background()
	bo := ratecontrol.NewExponentialBackoffOffset(time.Millisecond, 2,
		ratecontrol.WithUnlimitedRetries(true))

	retries := 5
	for i := range retries {
		done, err := bo.Wait(ctx, nil)
		if err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		if done || bo.Done() {
			t.Fatalf("retry %d: unexpectedly done, want an unlimited backoff to continue", i)
		}
	}
	if got, want := bo.Retries(), retries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBackoffNextContextCancel(t *testing.T) {
	eb := ratecontrol.NewExponentialBackoff(time.Hour, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	select {
	case <-eb.Next():
		t.Error("expected the canceled context to win the select")
	case <-ctx.Done():
	}
}

func TestBackoffWaitDone(t *testing.T) {
	ctx := context.Background()
	numRetries := 2
	eb := ratecontrol.NewExponentialBackoff(time.Millisecond, numRetries)
	for {
		done, err := eb.Wait(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		if eb.Done() {
			t.Error("expected Done to be false while Wait returns false")
		}
	}
	if !eb.Done() {
		t.Error("expected Done after Wait has returned true")
	}
	if got, want := eb.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNoBackoff(t *testing.T) {
	nb := ratecontrol.NoBackoff{}
	for range 3 {
		select {
		case _, ok := <-nb.Next():
			if !ok {
				t.Fatal("expected ok to be true for NoBackoff.Next")
			}
		case <-time.After(time.Second):
			t.Fatal("Next did not fire immediately")
		}
		if nb.Done() {
			t.Error("NoBackoff must never be done")
		}
	}
	done, err := nb.Wait(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if done {
		t.Error("NoBackoff.Wait must return false")
	}
	if got, want := nb.Retries(), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
