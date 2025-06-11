// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made and for implementing backoff mechanisms.
package ratecontrol

import (
	"context"
	"sync/atomic"
	"time"
)

// Controller is used to control the rate at which requests are made and
// to implement backoff when the remote server is unwilling to process a
// request. Controller is safe to use concurrently.
type Controller struct {
	opts         options
	reqsTicker   *time.Ticker
	reqsPerTick  atomic.Int64
	bytesTicker  *time.Ticker
	bytesPerTick atomic.Int64
}

// New returns a new Controller configuring using the specified options.
func New(opts ...Option) *Controller {
	c := &Controller{}
	for _, fn := range opts {
		fn(&c.opts)
	}
	if c.opts.reqsPerTick > 0 {
		c.reqsTicker = time.NewTicker(c.opts.reqsInterval / time.Duration(c.opts.reqsPerTick))
	}
	if c.opts.bytesPerTick > 0 {
		c.bytesTicker = time.NewTicker(c.opts.bytesInterval)
	}
	return c
}

func (c *Controller) remaining(current *atomic.Int64, allowed int) bool {
	if allowed == 0 {
		return true
	}
	return current.Load() < int64(allowed)
}

func (c *Controller) waitBytesPerTick(ctx context.Context) error {
	if c.remaining(&c.bytesPerTick, c.opts.bytesPerTick) {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.bytesTicker.C:
		// reset the bytesPerTick counter.
		c.bytesPerTick.Store(0)
	}
	return nil
}

// Wait returns when a request can be made. Rate limiting of requests
// takes priority over rate limiting of bytes. That is, bytes are
// only considered when a new request can be made.
func (c *Controller) Wait(ctx context.Context) error {
	if c.opts.noRateControl {
		return nil
	}
	c.reqsPerTick.Add(1)
	if c.remaining(&c.reqsPerTick, c.opts.reqsPerTick) {
		return c.waitBytesPerTick(ctx)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.reqsTicker.C:
		c.reqsPerTick.Store(0)
		return c.waitBytesPerTick(ctx)
	}
}

// BytesTransferred notifies the controller that the specified number of bytes
// have been transferred and is used when byte based rate control is configured
// via WithBytesPerTick.
func (c *Controller) BytesTransferred(nBytes int) {
	if c.opts.bytesPerTick == 0 {
		return
	}
	c.bytesPerTick.Add(int64(nBytes))
}

func (c *Controller) Backoff() Backoff {
	if c.opts.noRateControl {
		return noBackoff{}
	}
	if c.opts.customBackoff != nil {
		return c.opts.customBackoff()
	}
	if c.opts.backoffStart == 0 {
		return noBackoff{}
	}
	return NewExpontentialBackoff(c.opts.backoffStart, c.opts.backoffSteps)
}
