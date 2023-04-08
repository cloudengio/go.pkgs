// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"context"
	"net/http"
	"time"
)

// Backoff represents the interface to a backoff algorithm.
type Backoff interface {
	// Wait implements a backoff algorithm. It returns true if the backoff
	// should be terminated, i.e. no more requests should be attempted.
	// The error returned is nil when the backoff algorithm has reached
	// its limit and will generally only be non-nil for an internal error
	// such as the context being cancelled.
	Wait(context.Context, *http.Response) (bool, error)

	// Retries returns the number of retries that the backoff aglorithm
	// has recorded, ie. the number of times that Backoff was called and
	// returned false.
	Retries() int
}

type ExponentialBackoff struct {
	steps     int
	retries   int
	nextDelay time.Duration
}

// NewExpontentialBackoff returns a instance of Backoff that implements
// an exponential backoff algorithm starting with the specified initial
// delay and continuing for the specified number of steps.
func NewExpontentialBackoff(initial time.Duration, steps int) Backoff {
	return &ExponentialBackoff{nextDelay: initial, steps: steps}
}

// Retries implements Backoff.
func (eb *ExponentialBackoff) Retries() int {
	return eb.retries
}

// Wait implements Backoff.
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ *http.Response) (bool, error) {
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

type noBackoff struct{}

func (nb noBackoff) Retries() int {
	return 0
}

func (nb noBackoff) Wait(_ context.Context, _ *http.Response) (bool, error) {
	return false, nil
}
