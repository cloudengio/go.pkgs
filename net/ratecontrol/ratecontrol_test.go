// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"cloudeng.io/net/ratecontrol"
)

func TestNoop(t *testing.T) {
	ctx := context.Background()
	c := ratecontrol.New()
	for range 100 {
		backoff := c.Backoff()
		if err := c.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		done, err := backoff.Wait(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := done, false; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func waitForRequests(ctx context.Context, t *testing.T, c *ratecontrol.Controller, n, b int) time.Duration {
	c.BytesTransferred(b)
	then := time.Now()
	for range n {
		if err := c.Wait(ctx); err != nil {
			t.Fatal(err)
		}
		c.BytesTransferred(b)
	}
	return time.Since(then)
}

// tighter lower bound than upper bound since the former
// will be due to clock granularity issues and the latter to
// a slow machine which is common on CI systems.
func bounds(d, b time.Duration) (lower, upper time.Duration) {
	return d - b, d + (2 * b)
}

func TestRequestRate(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 500
	c := ratecontrol.New(ratecontrol.WithRequestsPerTick(tick, 1))
	took := waitForRequests(ctx, t, c, 2, 0)
	lower, upper := bounds(2*tick, 50*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

func TestRequestRateConcurrent(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 500
	c := ratecontrol.New(ratecontrol.WithRequestsPerTick(tick, 2))
	var wg sync.WaitGroup
	wg.Add(4)
	then := time.Now()
	for _ = range 4 {
		go func() {
			waitForRequests(ctx, t, c, 2, 0)
			wg.Done()
		}()
	}
	wg.Wait()
	took := time.Since(then)
	lower, upper := bounds(tick*2, 200*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

func TestDataRateConcurrent(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 100
	c := ratecontrol.New(ratecontrol.WithBytesPerTick(tick, 10))
	var wg sync.WaitGroup
	wg.Add(4)
	then := time.Now()
	for range 4 {
		go func() {
			waitForRequests(ctx, t, c, 10, 10)
			wg.Done()
		}()
	}
	wg.Wait()
	took := time.Since(then)
	// 10 concurrent iterations requires 40 ticks to send 400 bytes.
	lower, upper := bounds(40*tick, 100*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

func TestDataRate(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 100
	c := ratecontrol.New(ratecontrol.WithBytesPerTick(tick, 10))
	took := waitForRequests(ctx, t, c, 10, 10)
	// 10 iterations requires 10 ticks to send 100 bytes.
	lower, upper := bounds(10*tick, 50*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

func TestDataAndReqRate(t *testing.T) {
	ctx := context.Background()
	reqTick := time.Millisecond * 1000
	dataTick := time.Millisecond * 100
	c := ratecontrol.New(
		ratecontrol.WithBytesPerTick(dataTick, 10),
	)
	took := waitForRequests(ctx, t, c, 10, 10)
	// 10 iterations requires 10 ticks to send 100 bytes.
	lower, upper := bounds(10*dataTick, 50*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}

	// A low request rate will lower the data rate.
	c = ratecontrol.New(
		ratecontrol.WithBytesPerTick(dataTick, 10),
		ratecontrol.WithRequestsPerTick(reqTick, 1),
	)
	tookLonger := waitForRequests(ctx, t, c, 10, 10)

	lower, upper = bounds(10*reqTick, 50*time.Millisecond)
	if got := tookLonger; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}

	// Data rate when only request rate is in effect:
	dr := 100.0 / float64(took)
	drExpected := 100.0 / float64(reqTick)
	drLower, drUpper := drExpected*.9, dr*1.2
	if got := dr; got < drLower || drExpected > drUpper {
		t.Errorf("datarate: %v not in range %v..%v", got, drLower, drUpper)
	}

	// Data rate when limited by both the request and data rates.
	drSlower := 100.0 / float64(tookLonger)
	drExpected = 100.0 / float64(10*reqTick)
	drLower, drUpper = drExpected*.9, dr*1.2
	if got := drSlower; got < drLower || drExpected > drUpper {
		t.Errorf("datarate: %v not in range %v..%v", got, drLower, drUpper)
	}

}

func backoff(ctx context.Context, t *testing.T, c *ratecontrol.Controller) int {
	backoff := c.Backoff()
	for {
		done, err := backoff.Wait(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			return backoff.Retries()
		}
	}
}

func TestBackoff(t *testing.T) {
	ctx := context.Background()
	numRetries := 10
	c := ratecontrol.New(ratecontrol.WithExponentialBackoff(time.Millisecond, numRetries))
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
	c := ratecontrol.New(
		ratecontrol.WithExponentialBackoff(time.Hour, 10),
		ratecontrol.WithBytesPerTick(time.Second, 10),
		ratecontrol.WithRequestsPerTick(time.Second, 1),
	)
	go cancel()
	err := c.Wait(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}

	ctx, cancel = context.WithCancel(rootCtx)
	go cancel()
	last, err := c.Backoff().Wait(ctx, nil)

	if got, want := last, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}

	c = ratecontrol.New(ratecontrol.WithBytesPerTick(time.Second, 10))
	ctx, cancel = context.WithCancel(rootCtx)
	c.BytesTransferred(1000)
	go cancel()

	err = c.Wait(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}

type customBackoff struct {
	resp *http.Response
}

func (b *customBackoff) Wait(_ context.Context, resp any) (bool, error) {
	b.resp = resp.(*http.Response)
	return false, nil
}

func (b *customBackoff) Retries() int {
	return 33
}

func TestCustomBackoff(t *testing.T) {
	ctx := context.Background()
	backoff := &customBackoff{}
	resp := &http.Response{}

	c := ratecontrol.New(
		ratecontrol.WithCustomBackoff(func() ratecontrol.Backoff {
			return backoff
		}),
	)
	_, _ = c.Backoff().Wait(ctx, resp)
	if got, want := backoff.resp, resp; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
