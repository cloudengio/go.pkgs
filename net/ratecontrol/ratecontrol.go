// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made and for implementing backoff mechanisms.
package ratecontrol

import (
	"context"
	"sync"
	"time"
)

// Controller is used to control the rate at which requests are made and
// to implement backoff when the remote server is unwilling to process a
// request. Controller is safe to use concurrently.
type Controller struct {
	opts            options
	mu              sync.Mutex
	ticker          *time.Ticker
	curTick         int
	curBytesPerTick int
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
	return c
}

// updateBytesPerTick updates the current bytes per tick value and returns
// true if the current rate is within bounds and hence the caller need not
// wait.
func (c *Controller) updateBytesPerTick() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctick := c.opts.clock.Tick()
	if ctick != c.curTick {
		c.curTick = ctick
		c.curBytesPerTick = 0
		return true
	}
	if c.curBytesPerTick <= c.opts.bytesPerTick {
		return true
	}
	return false
}

func (c *Controller) waitBytesPerTick(ctx context.Context) error {
	if c.opts.bytesPerTick == 0 {
		return nil
	}
	if c.updateBytesPerTick() {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.opts.clock.after(c.opts.clock.TickDuration()):
	}
	return nil
}

// BytesTransferred notifies the controller that the specified number of bytes
// have been transferred and is used when byte based rate control is configured
// via WithBytesPerTick.
func (c *Controller) BytesTransferred(nBytes int) {
	if c.opts.bytesPerTick == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
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

func (c *Controller) Backoff() Backoff {
	if c.opts.backoffStart == 0 {
		return noBackoff{}
	}
	return NewExpontentialBackoff(c.opts.clock, c.opts.backoffStart, c.opts.backoffSteps)
}
