// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made.
package ratecontrol

import "time"

// Clock represents a clock used for rate limiting.
type Clock interface {
	Tick() int
	TickDuration() time.Duration

	// for testing.
	after(d time.Duration) <-chan time.Time
}

type clock struct{}

// Tick returns the current tick which will increment every TickDuration()
func (c clock) Tick() int {
	return time.Now().Minute()
}

// TickDuration returns the duration of a tick.
func (c clock) TickDuration() time.Duration {
	return time.Minute
}

// after returns a channel as per timer.After.
func (c clock) after(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Option represents an option for configuring a ratecontrol Controller.
type Option func(c *options)

// WithRequestsPerTick sets the rate for download requests in requests
// per tick, where tick is the smallest unit of time reported by the
// Clock implementation in use. The default clock uses time.Now().Minute()
// as the interval for rate limiting and hence the rate is in requests per
// minute.
func WithRequestsPerTick(rpt int) Option {
	return func(o *options) {
		o.reqsPerTick = rpt
	}
}

// WithBytesPerTick sets the approximate rate for downloads in bytes per tick,
// where tick is the smallest unit of time reported by the Clock implementation
// in use. The default clock uses time.Now().Minute() and hence the rate
// is in bytes per minute. The aglorithm used is very simply and will
// wait for one tick if the limit is reached without taking into account
// how long the tick is, nor how much excess data was sent over the
// previous tick (ie. no attempt is made to smooth out the rate and for now
// it's a simple start/stop model).
func WithBytesPerTick(bpt int) Option {
	return func(o *options) {
		o.bytesPerTick = bpt
	}
}

// WithBackoffParameters enables an exponential backoff algorithm that
// is triggered when the download fails in a way that is retryable. The
// container (fs.FS) implementation must return an error that returns
// true for errors.Is(err, retryErr). First defines the first backoff delay,
// which is then doubled for every consecutive matching error until the
// download either succeeds or the specified number of steps (attempted
// downloads) is exceeded (the download is then deemed to have failed).
func WithBackoffParameters(first time.Duration, steps int) Option {
	return func(o *options) {
		o.backoffStart = first
		o.backoffSteps = steps
	}
}

// WithClock sets the clock implementation to use.
func WithClock(c Clock) Option {
	return func(o *options) {
		o.clock = c
	}
}

type options struct {
	reqsPerTick  int
	bytesPerTick int
	backoffStart time.Duration
	backoffSteps int
	clock        Clock
}
