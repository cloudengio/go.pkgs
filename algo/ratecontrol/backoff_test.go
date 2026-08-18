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
		case <-nb.Next():
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
