// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made and for backing off when the remote
// service is unwilling to process requests.
// DEPRECATED: This package has been moved to cloudeng.io/algo/ratecontrol;
// use that instead.
package ratecontrol

import (
	"time"

	"cloudeng.io/algo/ratecontrol"
)

// Option represents an option for configuring a ratecontrol Controller.
type Option ratecontrol.Option

// WithRequestsPerTick sets the rate for requests in requests per tick.
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option {
	return Option(ratecontrol.Option(ratecontrol.WithRequestsPerTick(tickInterval, rpt)))
}

// WithBytesPerTick sets the approximate rate in bytes per tick
// The algorithm used is very simple and will simply stop sending data
// wait for a single tick if the limit is reached without taking into account
// how long the tick is, nor how much excess data was sent over the
// previous tick (ie. no attempt is made to smooth out the rate and for now
// it's a simple start/stop model). The bytes to be accounted for are
// reported to the Controller via its BytesTransferred method.
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option {
	return Option(ratecontrol.Option(ratecontrol.WithBytesPerTick(tickInterval, bpt)))
}

// WithExponentialBackoff enables an exponential backoff algorithm.
// First defines the first backoff delay, which is then doubled for every
// consecutive retry until the download either succeeds or the specified
// number of steps (attempted requests) is exceeded.
func WithExponentialBackoff(first time.Duration, steps int) Option {
	return Option(ratecontrol.Option(ratecontrol.WithExponentialBackoff(first, steps, false)))
}

// WithCustomBackoff allows the use of a custom backoff function.
func WithCustomBackoff(backoff func() Backoff) Option {
	return Option(ratecontrol.Option(ratecontrol.WithBackoff(func() ratecontrol.Backoff {
		return ratecontrol.Backoff(backoff())
	})))
}

// WithNoRateControl creates a Controller that returns immediately
// and offers no backoff. It can be used as a default when no
// rate control is desired.
func WithNoRateControl() Option {
	return Option(ratecontrol.Option(ratecontrol.WithNoRateControl()))
}
