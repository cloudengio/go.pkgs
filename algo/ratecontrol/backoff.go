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

	// Next returns a channel, as per time.Timer.C, that is ready when the
	// next backoff delay has expired. Once the backoff algorithm has
	// reached its limit, Next returns a closed channel and Done will
	// return true. Note that this differs from a time.Timer.C, which is
	// never closed: receiving from the closed channel yields a zero
	// time.Time immediately (and repeatedly), so completion can be
	// detected either by calling Done or by testing the received value:
	//
	//  if _, ok := <-backoff.Next(); !ok {
	//    // backoff has reached its limit
	//  }
	//
	// This allows for the backoff to be used in select statements, as per
	// the following example:
	//
	//  for {
	//    select {
	//    ...
	//    case <-ctx.Done():
	//      return ctx.Err()
	//    case _, ok := <-backoff.Next():
	//      if !ok { // equivalently: backoff.Done()
	//        return err
	//      }
	//      ...
	//    }
	//  }
	//
	// The caller cannot stop the timer underlying the returned channel;
	// abandoned timers are garbage collected.
	Next() <-chan time.Time
	// Done returns true if the backoff algorithm has reached its limit and
	// no more requests should be attempted. The limit is reached when Wait
	// returns true or when Next is called after all retries have been used.
	Done() bool

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
	done      bool
}

// closedTimeChan is returned by Next once a backoff has reached its limit;
// receiving from it never blocks.
var closedTimeChan = func() <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}()

// NewExponentialBackoff returns a instance of ExponentialBackoff.
// If initial is less than or equal to zero, DefaultBackoffInterval is used.
// If steps is less than or equal to zero, DefaultBackoffSteps is used.
func NewExponentialBackoff(initial time.Duration, steps int) *ExponentialBackoff {
	if initial <= 0 {
		initial = DefaultBackoffInterval
	}
	if steps <= 0 {
		steps = DefaultBackoffSteps
	}
	return &ExponentialBackoff{nextDelay: initial, steps: steps}
}

// Retries implements Backoff.
func (eb *ExponentialBackoff) Retries() int {
	return eb.retries
}

// Wait implements Backoff.
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ any) (bool, error) {
	if eb.retries >= eb.steps {
		eb.done = true
		return true, nil
	}
	timer := time.NewTimer(eb.nextDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-timer.C:
	}
	if eb.nextDelay < time.Duration(1<<62) {
		eb.nextDelay *= 2
	}
	eb.retries++
	return false, nil
}

// Next implements Backoff. The retry is recorded when the timer is armed,
// not when it fires, so a caller that abandons the returned channel (eg.
// because its context was canceled) will still have consumed that retry.
func (eb *ExponentialBackoff) Next() <-chan time.Time {
	if eb.retries >= eb.steps {
		eb.done = true
		return closedTimeChan
	}
	delay := eb.nextDelay
	if eb.nextDelay < time.Duration(1<<62) {
		eb.nextDelay *= 2
	}
	eb.retries++
	return time.NewTimer(delay).C
}

// Done implements Backoff.
func (eb *ExponentialBackoff) Done() bool {
	return eb.done
}

// NoBackoff implements a Backoff that does not perform any backoff and always
// returns false for Wait and Done, an immediately ready channel for Next and
// 0 for Retries.
type NoBackoff struct{}

func (nb NoBackoff) Retries() int {
	return 0
}

func (nb NoBackoff) Wait(_ context.Context, _ any) (bool, error) {
	return false, nil
}

func (nb NoBackoff) Next() <-chan time.Time {
	return closedTimeChan
}

func (nb NoBackoff) Done() bool {
	return false
}

// ExponentialBackoffOffset implements an exponential backoff algorithm with
// a random offset used for the first delay, all subsequent delays
// are calculated as in ExponentialBackoff. The first delay is
// a random value between 0 and the initial delay.
type ExponentialBackoffOffset struct {
	ExponentialBackoff
}

// NewExponentialBackoffOffset returns a instance of ExponentialBackoffOffset.
// If initial is less than or equal to zero, DefaultBackoffInterval is used.
// If steps is less than or equal to zero, DefaultBackoffSteps is used.
func NewExponentialBackoffOffset(initial time.Duration, steps int) *ExponentialBackoffOffset {
	if initial <= 0 {
		initial = DefaultBackoffInterval
	}
	if steps <= 0 {
		steps = DefaultBackoffSteps
	}
	return &ExponentialBackoffOffset{
		ExponentialBackoff: ExponentialBackoff{nextDelay: initial, steps: steps},
	}
}

// randomOffset returns a random duration in (0, limit).
func randomOffset(limit time.Duration) time.Duration {
	offset := time.Duration(rand.Int63n(int64(limit))) //nolint:gosec // G404: false positive, no need for crypto strength randomness here.
	if offset == 0 {
		offset = time.Nanosecond
	}
	return offset
}

func (eb *ExponentialBackoffOffset) Wait(ctx context.Context, v any) (bool, error) {
	if eb.retries >= eb.steps {
		eb.done = true
		return true, nil
	}
	if eb.retries == 0 && eb.nextDelay > 0 {
		timer := time.NewTimer(randomOffset(eb.nextDelay))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-timer.C:
		}
		if eb.nextDelay < time.Duration(1<<62) {
			eb.nextDelay *= 2
		}
		eb.retries++
		return false, nil
	}
	return eb.ExponentialBackoff.Wait(ctx, v)
}

// Next implements Backoff, using a random offset for the first delay as
// per Wait.
func (eb *ExponentialBackoffOffset) Next() <-chan time.Time {
	if eb.retries == 0 && eb.retries < eb.steps && eb.nextDelay > 0 {
		offset := randomOffset(eb.nextDelay)
		if eb.nextDelay < time.Duration(1<<62) {
			eb.nextDelay *= 2
		}
		eb.retries++
		return time.NewTimer(offset).C
	}
	return eb.ExponentialBackoff.Next()
}
