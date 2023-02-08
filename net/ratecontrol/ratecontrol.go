// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made.
package ratecontrol

import (
	"context"
	"time"
)

// Controller is used to control the rate at which requests are made and
// to implement backoff when the remote server responds with a retriable
// error.
type Controller struct {
	opts             options
	ticker           *time.Ticker
	retries          int
	nextBackoffDelay time.Duration
	curTick          int
	curBytesPerTick  int
}

// New returns a new Controller configuring using the specified options.
func New(opts ...Option) *Controller {
	c := &Controller{}
	c.opts.clock = clock{}
	for _, fn := range opts {
		fn(&c.opts)
	}
	if c.opts.reqsPerTick > 0 {
		c.ticker = time.NewTicker(c.opts.clock.TickDuration() / time.Duration(c.opts.reqsPerTick))
	}
	if c.opts.bytesPerTick > 0 {
		c.curTick = c.opts.clock.Tick()
	}
	c.InitBackoff()
	return c
}

func (c *Controller) waitBytesPerTick(ctx context.Context) error {
	if c.opts.bytesPerTick == 0 {
		return nil
	}
	ctick := c.opts.clock.Tick()
	if ctick != c.curTick {
		c.curTick = ctick
		c.curBytesPerTick = 0
		return nil
	}
	if c.curBytesPerTick <= c.opts.bytesPerTick {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.opts.clock.after(c.opts.clock.TickDuration()):
	}
	return nil
}

// Received notifies the controller that the specified number of bytes
// have been received.
func (c *Controller) Received(nBytes int) {
	if c.opts.bytesPerTick == 0 {
		return
	}
	ctick := c.opts.clock.Tick()
	if ctick == c.curTick {
		c.curBytesPerTick += nBytes
		return
	}
	c.curTick = ctick
	c.curBytesPerTick = 0
}

// Wait returns when a request can be made. Rate limiting of requests
// takes priority over rate limiting of bytes. That is, bytes are
// only considered when a new request can be made.
func (c *Controller) Wait(ctx context.Context) error {
	if c.ticker == nil || c.ticker.C == nil {
		return c.waitBytesPerTick(ctx)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ticker.C:
		return c.waitBytesPerTick(ctx)
	}
}

// InitBackoff resets the backoff state ready for a new request.
func (c *Controller) InitBackoff() {
	c.retries = 0
	c.nextBackoffDelay = c.opts.backoffStart
}

// Retries the number of retries that have been performed.
func (c *Controller) Retries() int {
	return c.retries
}

// Backoff implements the backoff algorithm and will wait the appropriate
// amount of time before a retry is appropriate. It will return true
// when no more retries should be attempted.
func (c *Controller) Backoff(ctx context.Context) (bool, error) {
	if c.retries >= c.opts.backoffSteps {
		return true, nil
	}
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-c.opts.clock.after(c.nextBackoffDelay):
	}
	c.nextBackoffDelay *= 2
	c.retries++
	return false, nil
}
