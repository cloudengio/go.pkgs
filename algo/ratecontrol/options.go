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

const (
	DefaultTickInterval    = time.Second
	DefaultRequestsPerTick = 1
	DefaultBytesPerTick    = 1024 * 1024
	DefaultBackoffInterval = time.Second
	DefaultBackoffSteps    = 10
)

// WithRequestsPerTick sets the rate for requests in requests per tick.
// If tickInterval is less than or equal to zero, DefaultTickInterval is used.
// If rpt is less than or equal to zero, DefaultRequestsPerTick is used.
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option {
	return func(o *options) {
		if tickInterval <= 0 {
			tickInterval = DefaultTickInterval
		}
		if rpt <= 0 {
			rpt = DefaultRequestsPerTick
		}
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
// If tickInterval is less than or equal to zero, DefaultTickInterval is used.
// If bpt is less than or equal to zero, DefaultBytesPerTick is used.
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option {
	return func(o *options) {
		if tickInterval <= 0 {
			tickInterval = DefaultTickInterval
		}
		if bpt <= 0 {
			bpt = DefaultBytesPerTick
		}
		o.bytesInterval = tickInterval
		o.bytesPerTick = bpt
	}
}

// WithExponentialBackoff enables an exponential backoff algorithm.
// If randomizedOffset is false NewExponentialBackoff is used, otherwise
// NewExponentialBackoffOffset is used.
// If first is less than or equal to zero, DefaultBackoffInterval is used.
// If steps is less than or equal to zero, DefaultBackoffSteps is used.
func WithExponentialBackoff(first time.Duration, steps int, randomizedOffset bool) Option {
	if first <= 0 {
		first = DefaultBackoffInterval
	}
	if steps <= 0 {
		steps = DefaultBackoffSteps
	}
	return func(o *options) {
		if randomizedOffset {
			o.backoff = func() Backoff {
				return NewExponentialBackoffOffset(first, steps)
			}
			return
		}
		o.backoff = func() Backoff { return NewExponentialBackoff(first, steps) }
	}
}

// WithBackoff allows the use of a custom backoff function.
func WithBackoff(backoff func() Backoff) Option {
	return func(o *options) {
		o.backoff = backoff
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
	backoff       func() Backoff
}
