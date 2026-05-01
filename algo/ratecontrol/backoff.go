// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"context"
	"math/rand"
	"time"
)

// Backoff represents the interface to a backoff algorithm.
type Backoff interface {
	// Wait implements a backoff algorithm. It returns true if the backoff
	// should be terminated, i.e. no more requests should be attempted.
	// The error returned is nil when the backoff algorithm has reached
	// its limit and will generally only be non-nil for an internal error
	// such as the context being canceled.
	// The second argument is a placeholder for any additional data that
	// the backoff algorithm may need to process, such as an HTTP response
	// or a retry response. It can be nil if no such data is needed.
	Wait(context.Context, any) (bool, error)

	// Retries returns the number of retries that the backoff algorithm
	// has recorded, ie. the number of times that Backoff was called and
	// returned false.
	Retries() int
}

// ExponentialBackoff implements an exponential backoff algorithm. It starts
// with the specified initial delay and doubles the delay for each retry up to
// the specified number of steps.
type ExponentialBackoff struct {
	steps     int
	retries   int
	nextDelay time.Duration
}

func NewExponentialBackoff(initial time.Duration, steps int) *ExponentialBackoff {
	return &ExponentialBackoff{nextDelay: initial, steps: steps}
}

// Retries implements Backoff.
func (eb *ExponentialBackoff) Retries() int {
	return eb.retries
}

// Wait implements Backoff.
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ any) (bool, error) {
	if eb.retries >= eb.steps {
		return true, nil
	}
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-time.After(eb.nextDelay):
	}
	eb.nextDelay *= 2
	eb.retries++
	return false, nil
}

// NoBackoff implements a Backoff that does not perform any backoff and always
// returns false for Wait and 0 for Retries.
type NoBackoff struct{}

func (nb NoBackoff) Retries() int {
	return 0
}

func (nb NoBackoff) Wait(_ context.Context, _ any) (bool, error) {
	return false, nil
}

// ExponentialBackoffOffset implements an exponential backoff algorithm with
// a random offset used for the first delay, all subsequent delays
// are calculated as in ExponentialBackoff. The first delay is
// a random value between 0 and the initial delay.
type ExponentialBackoffOffset struct {
	ExponentialBackoff
}

func NewExponentialBackoffOffset(initial time.Duration, steps int) *ExponentialBackoffOffset {
	return &ExponentialBackoffOffset{
		ExponentialBackoff: ExponentialBackoff{nextDelay: initial, steps: steps},
	}
}

func (eb *ExponentialBackoffOffset) Wait(ctx context.Context, _ any) (bool, error) {
	if eb.retries >= eb.steps {
		return true, nil
	}
	if eb.retries == 0 && eb.nextDelay > 0 {
		src := rand.NewSource(time.Now().UnixNano())
		eb.nextDelay = time.Duration(rand.New(src).Int63n(int64(eb.nextDelay)))
	}
	return eb.ExponentialBackoff.Wait(ctx, nil)
}
