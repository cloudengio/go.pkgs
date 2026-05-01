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
// Call Stop to free up resources when the Controller is no longer needed.
// The controller attempts to implement a smooth rate of requests and bytes\
// over the specified tick intervals.
type Controller struct {
	opts         options
	reqsTicker   *time.Ticker
	reqsBurst    atomic.Int64 // remaining burst tokens; allows first reqsPerTick requests to skip the ticker
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
			interval = time.Nanosecond
		}
		c.reqsBurst.Store(int64(c.opts.reqsPerTick))
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
	// Snapshot the broadcast channel before blocking. If the reset fires
	// between here and the select, ch will already be closed and the select
	// returns immediately.
	c.bytesMu.Lock()
	ch := c.bytesReset
	c.bytesMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.bytesTicker.C:
		// Won the tick: reset the counter and wake all other waiters.
		c.bytesPerTick.Store(0)
		c.bytesMu.Lock()
		old := c.bytesReset
		c.bytesReset = make(chan struct{})
		c.bytesMu.Unlock()
		close(old)
	case <-ch:
		// Woken by broadcast from the goroutine that won the tick.
	}
	// Proceed regardless of the current counter value. Re-checking here would
	// re-serialize goroutines: the winner's BytesTransferred call can push the
	// counter back to the limit before other waiters get a chance to check.
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
		if c.reqsBurst.Add(-1) < 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.reqsTicker.C:
			}
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

// Backoff returns an instance of the configured backoff algorithm. If no backoff algorithm is configured NoBackoff is returned.
func (c *Controller) Backoff() Backoff {
	if c.opts.noRateControl {
		return NoBackoff{}
	}
	if c.opts.backoff != nil {
		return c.opts.backoff()
	}
	return NoBackoff{}
}

// Stop stops the Controller's tickers. It should be called when the Controller
// is no longer needed to release resources.
func (c *Controller) Stop() {
	if c.reqsTicker != nil {
		c.reqsTicker.Stop()
	}
	if c.bytesTicker != nil {
		c.bytesTicker.Stop()
	}
}
