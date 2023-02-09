// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made and for backing off when the remote
// service is unwilling to process requests.
package ratecontrol

import "time"

// Clock represents a clock used for rate limiting. It determines the current
// time in 'ticks' and the wall-clock duration of a tick. This allows for
// rates to be specified in terms of requests per tick or bytes per tick rather
// than over a fixed duration. A default Clock implementation is provided
// which uses time.Minute as the tick duration.
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

// after returns a channel as per timer.After, it is used for testing.
func (c clock) after(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Option represents an option for configuring a ratecontrol Controller.
type Option func(c *options)

// WithRequestsPerTick sets the rate for requests in requests per tick,
// where tick is the unit of time reported by the Clock implementation in use.
// The default clock uses time.Now().Minute() as the interval for rate limiting
// and hence the rate is in requests per minute.
func WithRequestsPerTick(rpt int) Option {
	return func(o *options) {
		o.reqsPerTick = rpt
	}
}

// WithBytesPerTick sets the approximate rate in bytes per tick, where
// tick is the unit of time reported by the Clock implementation
// in use. The default clock uses time.Now().Minute() and hence the rate
// is in bytes per minute. The algorithm used is very simple and will
// wait for a single tick if the limit is reached without taking into account
// how long the tick is, nor how much excess data was sent over the
// previous tick (ie. no attempt is made to smooth out the rate and for now
// it's a simple start/stop model). The bytes to be accounted for are
// reported to the Controller via its BytesTransferred method.
func WithBytesPerTick(bpt int) Option {
	return func(o *options) {
		o.bytesPerTick = bpt
	}
}

// WithExponentialBackoff enables an exponential backoff algorithm.
// First defines the first backoff delay, which is then doubled for every
// consecutive retry until the download either succeeds or the specified
// number of steps (attempted requests) is exceeded.
func WithExponentialBackoff(first time.Duration, steps int) Option {
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
