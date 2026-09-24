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
	// Next records a retry and arms a new timer on each invocation. It is
	// intended to be called when a retry is needed, for example in a select
	// statement:
	//
	//  for {
	//    if err := doOperation(); err != nil {
	//      select {
	//      case <-ctx.Done():
	//        return ctx.Err()
	//      case _, ok := <-backoff.Next():
	//        if !ok { // equivalently: backoff.Done()
	//          return err
	//        }
	//      }
	//      continue
	//    }
	//    return nil
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
// the specified number of steps, ie. the largest delay it uses is
// initial * 2^(steps-1). See WithRandomizedOffset and WithUnlimitedRetries for
// the available variations on that behaviour.
type ExponentialBackoff struct {
	steps            int
	retries          int
	nextDelay        time.Duration
	done             bool
	randomizedOffset bool
	unlimitedRetries bool
}

// ExponentialBackoffOption represents an option to NewExponentialBackoff and
// NewExponentialBackoffOffset.
type ExponentialBackoffOption func(*exponentialBackoffOptions)

type exponentialBackoffOptions struct {
	randomizedOffset bool
	unlimitedRetries bool
}

// WithRandomizedOffset controls whether a random duration in (0, initial) is
// used for the first delay; all subsequent delays are calculated as usual
// either way. Randomizing it spreads the first retry of many clients that
// start backing off at the same time over the initial interval, so as to
// avoid a thundering herd, and is what NewExponentialBackoffOffset applies.
// It is disabled by default, so passing false is only useful to override an
// option applied earlier, such as that one.
func WithRandomizedOffset(v bool) ExponentialBackoffOption {
	return func(o *exponentialBackoffOptions) {
		o.randomizedOffset = v
	}
}

// WithUnlimitedRetries controls whether the backoff continues indefinitely.
// When it does, once steps retries have been recorded the delay stops doubling
// and every retry from then on uses that maximum delay, ie.
// initial * 2^(steps-1), and Wait and Done never return true, so terminating
// the backoff is left entirely to the caller, eg. by canceling the context
// passed to Wait. It is disabled by default, ie. the backoff is done once
// steps retries have been recorded, so passing false is only useful to
// override an option applied earlier.
func WithUnlimitedRetries(v bool) ExponentialBackoffOption {
	return func(o *exponentialBackoffOptions) {
		o.unlimitedRetries = v
	}
}

// maxDoublableDelay is the largest delay that can be doubled without
// overflowing a time.Duration.
const maxDoublableDelay = time.Duration(1 << 62)

// closedTimeChan is returned by Next once a backoff has reached its limit;
// receiving from it never blocks.
var closedTimeChan = func() <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}()

// NewExponentialBackoff returns a instance of ExponentialBackoff, configured by
// the supplied options (see WithRandomizedOffset and WithUnlimitedRetries).
// If initial is less than or equal to zero, DefaultBackoffInterval is used.
// If steps is less than or equal to zero, DefaultBackoffSteps is used.
func NewExponentialBackoff(initial time.Duration, steps int, opts ...ExponentialBackoffOption) *ExponentialBackoff {
	if initial <= 0 {
		initial = DefaultBackoffInterval
	}
	if steps <= 0 {
		steps = DefaultBackoffSteps
	}
	var o exponentialBackoffOptions
	for _, fn := range opts {
		fn(&o)
	}
	return &ExponentialBackoff{
		nextDelay:        initial,
		steps:            steps,
		randomizedOffset: o.randomizedOffset,
		unlimitedRetries: o.unlimitedRetries,
	}
}

// peekDelay returns the delay to use for the next retry and whether the backoff
// has reached its limit, without recording the retry: Wait records one only
// once the delay has elapsed, whereas Next records one as soon as the timer is
// armed.
func (eb *ExponentialBackoff) peekDelay() (time.Duration, bool) {
	if eb.retries >= eb.steps && !eb.unlimitedRetries {
		eb.done = true
		return 0, true
	}
	if eb.retries == 0 && eb.randomizedOffset && eb.nextDelay > 0 {
		return randomOffset(eb.nextDelay), false
	}
	return eb.nextDelay, false
}

// advance records a retry and doubles the delay used for the next one, except
// on the last step so that the delay is left pinned at its maximum for an
// unlimited backoff.
func (eb *ExponentialBackoff) advance() {
	if eb.retries+1 < eb.steps && eb.nextDelay < maxDoublableDelay {
		eb.nextDelay *= 2
	}
	eb.retries++
}

// Retries implements Backoff.
func (eb *ExponentialBackoff) Retries() int {
	return eb.retries
}

// Wait implements Backoff.
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ any) (bool, error) {
	delay, done := eb.peekDelay()
	if done {
		return true, nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-timer.C:
	}
	eb.advance()
	return false, nil
}

// Next implements Backoff. The retry is recorded when the timer is armed,
// not when it fires, so a caller that abandons the returned channel (eg.
// because its context was canceled) will still have consumed that retry.
func (eb *ExponentialBackoff) Next() <-chan time.Time {
	delay, done := eb.peekDelay()
	if done {
		return closedTimeChan
	}
	eb.advance()
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
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func (nb NoBackoff) Done() bool {
	return false
}

// ExponentialBackoffOffset implements an exponential backoff algorithm with
// a random offset used for the first delay, all subsequent delays
// are calculated as in ExponentialBackoff. The first delay is
// a random value between 0 and the initial delay. It is an ExponentialBackoff
// with WithRandomizedOffset applied.
type ExponentialBackoffOffset struct {
	*ExponentialBackoff
}

// NewExponentialBackoffOffset returns a instance of ExponentialBackoffOffset,
// ie. NewExponentialBackoff with WithRandomizedOffset(true) applied ahead of
// any options supplied here; since later options win, an explicit
// WithRandomizedOffset(false) overrides it.
// If initial is less than or equal to zero, DefaultBackoffInterval is used.
// If steps is less than or equal to zero, DefaultBackoffSteps is used.
func NewExponentialBackoffOffset(initial time.Duration, steps int, opts ...ExponentialBackoffOption) *ExponentialBackoffOffset {
	opts = append([]ExponentialBackoffOption{WithRandomizedOffset(true)}, opts...)
	return &ExponentialBackoffOffset{
		ExponentialBackoff: NewExponentialBackoff(initial, steps, opts...),
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
