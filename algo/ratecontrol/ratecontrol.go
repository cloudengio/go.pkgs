// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Limiter is an interface that defines a generic rate limiter.
type Limiter interface {
	Wait(context.Context) error
	BytesTransferred(int)
	Backoff() Backoff
}

// Controller implements Limiter and is used to control the rate at which
// requests are made and to implement backoff when the remote server is
// unwilling to process a request. Controller is safe to use concurrently.
type Controller struct {
	opts         options
	reqsTicker   *time.Ticker
	bytesTicker  *time.Ticker
	bytesPerTick atomic.Int64

	bytesMu    sync.Mutex
	bytesReset chan struct{} // closed and replaced on each interval reset to broadcast to all waiters
}

// New returns a new Controller configured using the specified options.
func New(opts ...Option) *Controller {
	c := &Controller{}
	for _, fn := range opts {
		fn(&c.opts)
	}
	if c.opts.reqsPerTick > 0 {
		interval := c.opts.reqsInterval / time.Duration(c.opts.reqsPerTick)
		if interval <= 0 {
			interval = time.Millisecond
		}
		c.reqsTicker = time.NewTicker(interval)
	}
	if c.opts.bytesPerTick > 0 {
		c.bytesTicker = time.NewTicker(c.opts.bytesInterval)
		c.bytesReset = make(chan struct{})
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
	// Snapshot the current broadcast channel before blocking so we wake
	// on the very next reset even if it happens between this read and the select.
	c.bytesMu.Lock()
	ch := c.bytesReset
	c.bytesMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.bytesTicker.C:
		// This goroutine won the tick: reset the counter and broadcast
		// to all other waiters by closing the current reset channel.
		c.bytesPerTick.Store(0)
		c.bytesMu.Lock()
		old := c.bytesReset
		c.bytesReset = make(chan struct{})
		c.bytesMu.Unlock()
		close(old)
	case <-ch:
		// Another goroutine already handled the tick and reset the counter.
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
	if c.opts.reqsPerTick > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.reqsTicker.C:
		}
	}
	return c.waitBytesPerTick(ctx)
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
		return NoBackoff{}
	}
	if c.opts.customBackoff != nil {
		return c.opts.customBackoff()
	}
	if c.opts.backoffStart == 0 {
		return NoBackoff{}
	}
	return NewExponentialBackoff(c.opts.backoffStart, c.opts.backoffSteps)
}
