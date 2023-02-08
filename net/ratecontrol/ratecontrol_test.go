// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"testing"
	"time"

	"cloudeng.io/net/ratecontrol"
)

func TestNoop(t *testing.T) {
	ctx := context.Background()
	clk := &ratecontrol.TestClock{}
	c := ratecontrol.New(ratecontrol.WithClock(clk))
	for i := 0; i < 100; i++ {
		if err := c.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		done, err := c.Backoff(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := done, true; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if got, want := clk.Called, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRequestRate(t *testing.T) {
	ctx := context.Background()
	clk := &ratecontrol.TestClock{TickValue: 10 * time.Millisecond}
	c := ratecontrol.New(ratecontrol.WithClock(clk),
		ratecontrol.WithRequestsPerTick(1))
	now := time.Now()
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 9)
		if err := c.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}
	since := time.Since(now)
	// tighter lower bound than upper bound since the former
	// will be due to clock granularity issues and the latter to
	// a slow machine which is common on CI systems.
	lower, upper := 90*time.Millisecond, 150*time.Millisecond
	if got := since; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
	if got, want := clk.Called, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDataRate(t *testing.T) {
	ctx := context.Background()
	clk := &ratecontrol.TestClock{AfterValue: time.Millisecond, TickValue: 10 * time.Millisecond}
	c := ratecontrol.New(ratecontrol.WithClock(clk),
		ratecontrol.WithBytesPerTick(10))
	for i := 0; i < 10; i++ {
		if err := c.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		c.BytesTransferred(100)
		if got, want := len(clk.AfterDurations), i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func backoff(ctx context.Context, t *testing.T, c *ratecontrol.Controller) int {
	c.InitBackoff()
	for {
		done, err := c.Backoff(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			return c.Retries()
		}
	}
}

func TestBackoff(t *testing.T) {
	ctx := context.Background()
	// return immediately on retry
	clk := &ratecontrol.TestClock{AfterValue: time.Nanosecond}
	numRetries := 10
	c := ratecontrol.New(ratecontrol.WithClock(clk),
		ratecontrol.WithBackoffParameters(time.Millisecond, numRetries),
	)
	for i := 0; i < 3; i++ {
		retries := backoff(ctx, t, c)
		if got, want := retries, numRetries; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestCancel(t *testing.T) {
	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)
	clk := &ratecontrol.TestClock{AfterValue: time.Hour, TickValue: time.Hour}
	c := ratecontrol.New(ratecontrol.WithClock(clk),
		ratecontrol.WithBackoffParameters(time.Hour, 10),
		ratecontrol.WithBytesPerTick(10),
		ratecontrol.WithRequestsPerTick(1),
	)
	go cancel()
	err := c.Wait(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}

	ctx, cancel = context.WithCancel(rootCtx)
	go cancel()
	last, err := c.Backoff(ctx)

	if got, want := last, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}

	c = ratecontrol.New(ratecontrol.WithClock(clk),
		ratecontrol.WithBytesPerTick(10))
	ctx, cancel = context.WithCancel(rootCtx)
	c.BytesTransferred(1000)
	go cancel()

	err = c.Wait(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}
