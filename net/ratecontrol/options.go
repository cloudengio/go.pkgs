// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made and for backing off when the remote
// service is unwilling to process requests.
package ratecontrol

import "time"

// Option represents an option for configuring a ratecontrol Controller.
type Option func(c *options)

// WithRequestsPerTick sets the rate for requests in requests per tick.
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option {
	return func(o *options) {
		o.reqsInterval = tickInterval
		o.reqsPerTick = rpt
	}
}

// WithBytesPerTick sets the approximate rate in bytes per tick
// The algorithm used is very simple and will simply stop sending data
// wait for a single tick if the limit is reached without taking into account
// how long the tick is, nor how much excess data was sent over the
// previous tick (ie. no attempt is made to smooth out the rate and for now
// it's a simple start/stop model). The bytes to be accounted for are
// reported to the Controller via its BytesTransferred method.
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option {
	return func(o *options) {
		o.bytesInterval = tickInterval
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

// WithCustomBackoff allows the use of a custom backoff function.
func WithCustomBackoff(backoff func() Backoff) Option {
	return func(o *options) {
		o.customBackoff = backoff
	}
}

// WithNoRateControl creates a Controller that returns immediately
// and offers no backoff. It can be used as a default when no
// rate control is desired.
func WithNoRateControl() Option {
	return func(o *options) {
		o.noRateControl = true
	}
}

type options struct {
	noRateControl bool // if true, no rate control is applied
	reqsInterval  time.Duration
	reqsPerTick   int
	bytesInterval time.Duration
	bytesPerTick  int
	backoffStart  time.Duration
	backoffSteps  int
	customBackoff func() Backoff
}
