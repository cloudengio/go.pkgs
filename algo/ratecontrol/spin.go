// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// SpinDetector can be used to detect when a caller is spinning. A spin
// is defined as an excess of some number of calls within a time period.
// Tick should be called on each iteration, and the detector will reset the count and
// expiration when a spin is detected.
// SpinDetectorOption configures a SpinDetector.
type SpinDetectorOption func(*SpinDetector)

// WithClock overrides the clock used by SpinDetector. Intended for testing.
func WithClock(f func() time.Time) SpinDetectorOption {
	return func(s *SpinDetector) { s.now = f }
}

type SpinDetector struct {
	max        int64
	count      atomic.Int64
	mu         sync.Mutex
	expiration time.Time
	period     time.Duration
	now        func() time.Time
}

func NewSpinDetector(maxIterations int64, period time.Duration, opts ...SpinDetectorOption) *SpinDetector {
	s := &SpinDetector{
		max:    maxIterations,
		period: period,
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.expiration = s.now().Add(period)
	return s
}

func (s *SpinDetector) Tick() bool {
	count := s.count.Add(1)
	if count < s.max {
		return false
	}
	// count has reached the threshold; take the lock to safely read and update
	// expiration. Both paths (spin detected and period expired) reset state so
	// the detector is ready for the next window.
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	spinning := now.Before(s.expiration)
	s.count.Store(0)
	s.expiration = now.Add(s.period)
	return spinning
}

// BackoffOnSpin is a backoff strategy that is invoked when a caller is detected
// to be spinning. It is intended to be used as a safeguard in situations where
// a caller expects to be able to make a call without a need for a backoff strategy,
// but that in rare cases a spin may occur due to unexpected conditions.
// In such cases, this strategy will apply a backoff to prevent excessive spinning.
// Note the backoff is stateful across detections of a spin.
type BackoffOnSpin struct {
	SpinDetector *SpinDetector
	Backoff      Backoff
}

func NewBackoffOnSpin(maxIterations int64, period time.Duration, backoff Backoff, opts ...SpinDetectorOption) *BackoffOnSpin {
	return &BackoffOnSpin{
		SpinDetector: NewSpinDetector(maxIterations, period, opts...),
		Backoff:      backoff,
	}
}

func (b *BackoffOnSpin) Wait(ctx context.Context, val any) (bool, error) {
	if b.SpinDetector.Tick() {
		return b.Backoff.Wait(ctx, val)
	}
	return false, nil
}

func (b *BackoffOnSpin) Retries() int {
	return b.Backoff.Retries()
}
