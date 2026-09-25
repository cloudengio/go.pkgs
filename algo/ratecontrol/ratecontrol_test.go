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

	"cloudeng.io/algo/ratecontrol"
	"cloudeng.io/sync/ctxsync"
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

// bounds returns the range that a measured duration may fall in when the
// intended duration is d, with b as the tolerance on the lower bound.
//
// The lower bound is tight, because timers never fire early: a measurement
// below it means requests were let through faster than the limiter was
// configured to allow. That is the failure these tests exist to catch, and
// every regression that matters here shows up that way -- a budget that is not
// enforced, or that is handed to each caller instead of shared between them,
// makes a run shorter rather than longer.
//
// The upper bound is deliberately very loose, since it can only ever be an
// upper bound on the machine rather than on the limiter. These tests measure
// wall-clock time, and one that is loaded, throttled or coalescing timers
// stretches every wait in the sequence. The stretching compounds: a caller is
// admitted at a tick boundary, so one that is stalled for long enough to miss
// the next boundary waits a whole extra tick, and a starved machine does that
// repeatedly. Overruns have been seen at 1.15x the intended duration
// (TestDataAndReqRate, on CI), 2.5x (TestRequestRateConcurrent) and 3.5x
// (TestControllerConfigNewControllerDataRate), though the limiter itself was
// measured consuming exactly one tick per Wait when run undisturbed, with and
// without the race detector. The upper bound is therefore a liveness check --
// the limiter is still making progress, rather than hung -- and not a measure
// of precision; the lower bound is what guards the rate.
func bounds(d, b time.Duration) (lower, upper time.Duration) {
	return d - b, max(10*d, d+5*time.Second)
}

func TestRequestRate(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 500
	c := ratecontrol.New(ratecontrol.WithRequestsPerTick(tick, 1))
	took := waitForRequests(ctx, t, c, 2, 0)
	// burst=1 makes the first Wait immediate; only the second blocks for one tick.
	lower, upper := bounds(tick, 50*time.Millisecond)
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
	for range 4 {
		go func() {
			waitForRequests(ctx, t, c, 2, 0)
			wg.Done()
		}()
	}
	wg.Wait()
	took := time.Since(then)
	// burst=2 makes the first 2 Waits immediate; the remaining 6 each block for 250ms (tick/2),
	// so total ≈ 6*(tick/2) = 3*tick.
	lower, upper := bounds(tick*3, 200*time.Millisecond)
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
	// The budget is enforced strictly across the 4 goroutines, so they share
	// one budget rather than getting one each: a tick admits a single Wait,
	// whose 10 bytes exhaust the budget again, and the 40 Waits therefore take
	// 40 ticks rather than the 10 they would take if each goroutine were
	// allowed its own 10 bytes per tick. The lower bound is what pins that
	// down: were the budget handed to each goroutine, this would finish in
	// roughly a quarter of the time.
	lower, upper := bounds(40*tick, 200*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

// TestDataRateFairness verifies that concurrent callers share the
// bytes-per-tick budget evenly: each tick admits the caller that has been
// waiting longest, so over many ticks every caller is admitted about the same
// number of times and none can monopolize the budget. A limiter that instead
// let waiters race for each tick would hand out a visibly uneven split over
// this many ticks.
func TestDataRateFairness(t *testing.T) {
	ctx := context.Background()
	tick := 20 * time.Millisecond
	const goroutines, ticks = 4, 60
	c := ratecontrol.New(ratecontrol.WithBytesPerTick(tick, 10))

	// Each goroutine records its own count, so the writes are to distinct
	// elements and are published by wg.Wait before they are read below.
	counts := make([]int, goroutines)
	deadline := time.Now().Add(time.Duration(ticks) * tick)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				if err := c.Wait(ctx); err != nil {
					t.Error(err)
					return
				}
				c.BytesTransferred(10)
				counts[i]++
			}
		}()
	}
	wg.Wait()

	lo, hi, total := counts[0], counts[0], 0
	for _, n := range counts {
		lo, hi = min(lo, n), max(hi, n)
		total += n
	}
	if total == 0 {
		t.Fatalf("no requests were admitted: %v", counts)
	}
	// Round-robin admission keeps every caller within a turn of the others; the
	// margin covers the initial requests admitted before the queue forms and
	// the one in flight per caller when the deadline passes.
	if spread := hi - lo; spread > 3 {
		t.Errorf("uneven split of the budget: %v (spread %v), want each caller admitted about equally", counts, spread)
	}
	// One admission per tick, so the total tracks the ticks that elapsed rather
	// than goroutines*ticks: the budget must not scale with the caller count.
	if maxTotal := ticks + 2*goroutines; total > maxTotal {
		t.Errorf("admitted %v requests in ~%v ticks, want at most %v", total, ticks, maxTotal)
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

	// burst=1 makes the first Wait immediate; the remaining 9 each block for
	// reqTick, ie. 9 real seconds strung together from 9 separate ticks. The
	// jitter that bounds' upper bound absorbs accumulates across all 9 of them
	// here, making this the coarsest of these checks; the tighter assertions on
	// the request rate are TestRequestRate and TestRequestRateConcurrent above,
	// whose much shorter durations leave the same absolute jitter a far smaller
	// fraction of the total.
	lower, upper = bounds(9*reqTick, 200*time.Millisecond)
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
	c := ratecontrol.New(ratecontrol.WithExponentialBackoff(time.Millisecond, numRetries, false))
	for range 3 {
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
		ratecontrol.WithExponentialBackoff(time.Hour, 10, false),
		ratecontrol.WithBytesPerTick(time.Second, 10),
		ratecontrol.WithRequestsPerTick(time.Second, 1),
	)
	// Exceed the byte limit so waitBytesPerTick blocks; the request burst would
	// otherwise return immediately, giving go cancel() no window to fire.
	c.BytesTransferred(100)
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

func (b *customBackoff) Next() <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func (b *customBackoff) Done() bool {
	return false
}

func TestCustomBackoff(t *testing.T) {
	ctx := context.Background()
	backoff := &customBackoff{}
	resp := &http.Response{}

	c := ratecontrol.New(
		ratecontrol.WithBackoff(func() ratecontrol.Backoff {
			return backoff
		}),
	)
	done, err := c.Backoff().Wait(ctx, resp)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := done, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := backoff.resp, resp; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	// Test with tickers initialized
	c := ratecontrol.New(
		ratecontrol.WithRequestsPerTick(time.Millisecond*10, 1),
		ratecontrol.WithBytesPerTick(time.Millisecond*10, 10),
	)
	c.Stop()
	c.Stop() // Test idempotency

	// Test concurrent Stop
	c2 := ratecontrol.New(
		ratecontrol.WithRequestsPerTick(time.Millisecond*10, 1),
		ratecontrol.WithBytesPerTick(time.Millisecond*10, 10),
	)
	var wg ctxsync.WaitGroup
	for range 10 {
		wg.Go(func() {
			c2.Stop()
		})
	}
	wg.Wait(ctx)

	// Test with no tickers initialized
	c3 := ratecontrol.New()
	c3.Stop()
}

// TestStopReleasesWaiters verifies that Stop releases callers already parked
// waiting for the bytes-per-tick budget, rather than leaving them blocked until
// a tick that will never come, and that a caller arriving afterwards is not
// parked at all.
func TestStopReleasesWaiters(t *testing.T) {
	ctx := context.Background()
	// A tick long enough that only Stop can release the waiters.
	c := ratecontrol.New(ratecontrol.WithBytesPerTick(time.Hour, 10))
	c.BytesTransferred(100) // exhaust the budget so that Wait parks

	const waiters = 3
	errs := make(chan error, waiters)
	for range waiters {
		go func() { errs <- c.Wait(ctx) }()
	}
	// Give them time to park, so that Stop is releasing parked waiters rather
	// than being seen by the fast path on the way in.
	time.Sleep(50 * time.Millisecond)
	c.Stop()

	for i := range waiters {
		select {
		case err := <-errs:
			if err == nil || err != context.Canceled {
				t.Errorf("waiter %v: got %v, want %v", i, err, context.Canceled)
			}
		case <-time.After(time.Minute):
			t.Fatalf("waiter %v was not released by Stop", i)
		}
	}

	// A caller arriving after Stop must also return rather than park.
	if err := c.Wait(ctx); err == nil || err != context.Canceled {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}
