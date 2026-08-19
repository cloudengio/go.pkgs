// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"math"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
)

func TestExponentialBackoffConfigNewBackoff(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  ratecontrol.ExponentialBackoffConfig
		want any
	}{
		{"exponential", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: 3}, &ratecontrol.ExponentialBackoff{}},
		{"randomized", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: 3, RandomizeStart: true}, &ratecontrol.ExponentialBackoffOffset{}},
		{"zero delay", ratecontrol.ExponentialBackoffConfig{Steps: 3}, ratecontrol.NoBackoff{}},
		{"zero steps", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond}, ratecontrol.NoBackoff{}},
		{"zero value", ratecontrol.ExponentialBackoffConfig{RandomizeStart: true}, ratecontrol.NoBackoff{}},
	} {
		got := tc.cfg.NewBackoff()
		if gt, wt := typeOf(got), typeOf(tc.want); gt != wt {
			t.Errorf("%v: got %v, want %v", tc.name, gt, wt)
		}
	}
}

func typeOf(v any) string {
	switch v.(type) {
	case *ratecontrol.ExponentialBackoff:
		return "*ExponentialBackoff"
	case *ratecontrol.ExponentialBackoffOffset:
		return "*ExponentialBackoffOffset"
	case ratecontrol.NoBackoff:
		return "NoBackoff"
	default:
		return "unknown"
	}
}

func TestExponentialBackoffConfigNewBackoffSteps(t *testing.T) {
	ctx := context.Background()
	numRetries := 3
	cfg := ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: numRetries}
	b := cfg.NewBackoff()
	for {
		done, err := b.Wait(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
	}
	if got, want := b.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := b.Done(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExponentialBackoffConfigTotalTimeout(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  ratecontrol.ExponentialBackoffConfig
		want time.Duration
	}{
		{"single step", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Second, Steps: 1}, time.Second},
		{"three steps", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: 3}, 7 * time.Millisecond},
		{"randomization ignored", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: 3, RandomizeStart: true}, 7 * time.Millisecond},
		{"zero delay", ratecontrol.ExponentialBackoffConfig{Steps: 3}, 0},
		{"zero steps", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond}, 0},
		{"overflow", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Hour, Steps: 64}, math.MaxInt64},
	} {
		if got, want := tc.cfg.TotalTimeout(), tc.want; got != want {
			t.Errorf("%v: got %v, want %v", tc.name, got, want)
		}
	}
}

func TestExponentialBackoffConfigString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  ratecontrol.ExponentialBackoffConfig
		want string
	}{
		{"exponential", ratecontrol.ExponentialBackoffConfig{InitialDelay: 100 * time.Millisecond, Steps: 3}, "initial=100ms steps=3 total=700ms"},
		{"randomized", ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Second, Steps: 2, RandomizeStart: true}, "initial=1s steps=2 total=3s randomized"},
		{"no backoff", ratecontrol.ExponentialBackoffConfig{Steps: 3}, "no backoff"},
	} {
		if got, want := tc.cfg.String(), tc.want; got != want {
			t.Errorf("%v: got %q, want %q", tc.name, got, want)
		}
	}
}

func TestRateConfigString(t *testing.T) {
	tick := time.Second
	for _, tc := range []struct {
		name string
		cfg  ratecontrol.RateConfig
		want string
	}{
		{"requests", ratecontrol.RateConfig{Tick: tick, RequestsPerTick: 10}, "10 requests/1s"},
		{"bytes", ratecontrol.RateConfig{Tick: tick, BytesPerTick: 1000}, "1000 bytes/1s"},
		{"both", ratecontrol.RateConfig{Tick: tick, RequestsPerTick: 10, BytesPerTick: 1000}, "10 requests/1s, 1000 bytes/1s"},
		{"none", ratecontrol.RateConfig{Tick: tick}, "no rate limits"},
	} {
		if got, want := tc.cfg.String(), tc.want; got != want {
			t.Errorf("%v: got %q, want %q", tc.name, got, want)
		}
	}
}

func TestExponentialBackoffConfigBackoffOption(t *testing.T) {
	cfg := ratecontrol.ExponentialBackoffConfig{InitialDelay: time.Millisecond, Steps: 3}
	c := ratecontrol.New(cfg.BackoffOption())
	if got, want := typeOf(c.Backoff()), "*ExponentialBackoff"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// Each call to Backoff must return a fresh instance.
	if c.Backoff() == c.Backoff() {
		t.Errorf("Backoff returned the same instance twice")
	}

	cfg.RandomizeStart = true
	c = ratecontrol.New(cfg.BackoffOption())
	if got, want := typeOf(c.Backoff()), "*ExponentialBackoffOffset"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Unlike NewBackoff, BackoffOption falls back to the package defaults
	// for zero values rather than disabling backoff.
	c = ratecontrol.New(ratecontrol.ExponentialBackoffConfig{}.BackoffOption())
	if got, want := typeOf(c.Backoff()), "*ExponentialBackoff"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestControllerConfigNewControllerDefaults(t *testing.T) {
	ctx := context.Background()
	c := ratecontrol.ControllerConfig{}.NewController()
	if got, want := typeOf(c.Backoff()), "NoBackoff"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// With no rate options configured, Wait must not block.
	c.BytesTransferred(1000)
	took := waitForRequests(ctx, t, c, 100, 10)
	if got, want := took, 100*time.Millisecond; got > want {
		t.Errorf("unconstrained Wait took %v, want < %v", got, want)
	}
}

func TestControllerConfigNewControllerBackoff(t *testing.T) {
	ctx := context.Background()
	numRetries := 3
	cfg := ratecontrol.ControllerConfig{
		ExponentialBackoff: ratecontrol.ExponentialBackoffConfig{
			InitialDelay: time.Millisecond,
			Steps:        numRetries,
		},
	}
	c := cfg.NewController()
	if got, want := typeOf(c.Backoff()), "*ExponentialBackoff"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	for range 3 {
		if got, want := backoff(ctx, t, c), numRetries; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	cfg.ExponentialBackoff.RandomizeStart = true
	c = cfg.NewController()
	if got, want := typeOf(c.Backoff()), "*ExponentialBackoffOffset"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestControllerConfigNewControllerRequestRate(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 500
	cfg := ratecontrol.ControllerConfig{
		Rate: ratecontrol.RateConfig{Tick: tick, RequestsPerTick: 1},
	}
	c := cfg.NewController()
	took := waitForRequests(ctx, t, c, 2, 0)
	// burst=1 makes the first Wait immediate; only the second blocks for one tick.
	lower, upper := bounds(tick, 50*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}

func TestControllerConfigNewControllerDataRate(t *testing.T) {
	ctx := context.Background()
	tick := time.Millisecond * 100
	cfg := ratecontrol.ControllerConfig{
		Rate: ratecontrol.RateConfig{Tick: tick, BytesPerTick: 10},
	}
	c := cfg.NewController()
	took := waitForRequests(ctx, t, c, 10, 10)
	// 10 iterations requires 10 ticks to send 100 bytes.
	lower, upper := bounds(10*tick, 50*time.Millisecond)
	if got := took; got < lower || got > upper {
		t.Errorf("wait delay: %v not in range %v..%v", got, lower, upper)
	}
}
