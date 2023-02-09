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
	opts             options
	mu               sync.Mutex
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

// InitBackoff resets the backoff state ready for a new request.
func (c *Controller) InitBackoff() {
	c.mu.Lock()
	c.retries = 0
	c.nextBackoffDelay = c.opts.backoffStart
	c.mu.Unlock()
}

// Retries the number of retries that have been performed.
func (c *Controller) Retries() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.retries
}

// Backoff implements an exponential backoff algorithm and will wait the
// appropriate amount of time before a retry is appropriate. It will return
// true when no more retries should be attempted (error is nil in this case).
func (c *Controller) Backoff(ctx context.Context) (bool, error) {
	c.mu.Lock()
	if c.retries >= c.opts.backoffSteps {
		c.mu.Unlock()
		return true, nil
	}
	backoffDelay := c.nextBackoffDelay
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-c.opts.clock.after(backoffDelay):
	}
	c.mu.Lock()
	c.nextBackoffDelay *= 2
	c.retries++
	c.mu.Unlock()
	return false, nil
}
