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
// The controller attempts to implement a smooth rate of requests and bytes
// over the specified tick intervals.
//
// Note that the tickers that pace requests and bytes are started lazily, on
// first use, rather than when New returns.
type Controller struct {
	opts       options
	reqsTokens chan struct{} // token bucket for request bursts

	reqsOnce   sync.Once
	reqsTicker *time.Ticker

	bytesOnce    sync.Once
	bytesTicker  *time.Ticker
	bytesPerTick atomic.Int64

	bytesMu    sync.Mutex
	bytesReset chan struct{} // closed and replaced on each interval reset to broadcast to all waiters

	stopOnce sync.Once
	stopCh   chan struct{}
}

// New returns a new Controller configured using the specified options.
func New(opts ...Option) *Controller {
	c := &Controller{
		stopCh: make(chan struct{}),
	}
	for _, fn := range opts {
		fn(&c.opts)
	}
	if c.opts.reqsPerTick > 0 {
		// The token bucket itself is not time sensitive -- an initial burst
		// being available immediately is the intended behaviour regardless
		// of when it is first drawn on -- so it is filled here. Only the
		// ticker that replenishes it after the burst is spent is started
		// lazily; see ensureReqsTicker.
		c.reqsTokens = make(chan struct{}, c.opts.reqsPerTick)
		for range c.opts.reqsPerTick {
			c.reqsTokens <- struct{}{}
		}
	}
	if c.opts.bytesPerTick > 0 {
		c.bytesReset = make(chan struct{})
	}
	return c
}

// ensureReqsTicker starts the ticker that replenishes the request token
// bucket, the first time it is needed. It is a no-op on every call after the
// first, and, once Stop has won the race to run reqsOnce's function instead
// (see Stop), forever after: reqsTicker is then left nil, which Wait never
// dereferences since it only ever reads from reqsTokens and stopCh.
func (c *Controller) ensureReqsTicker() {
	c.reqsOnce.Do(func() {
		interval := c.opts.reqsInterval / time.Duration(c.opts.reqsPerTick)
		if interval <= 0 {
			interval = time.Nanosecond
		}
		c.reqsTicker = time.NewTicker(interval)
		go func() {
			for {
				select {
				case <-c.reqsTicker.C:
					select {
					case c.reqsTokens <- struct{}{}:
					default:
					}
				case <-c.stopCh:
					return
				}
			}
		}()
	})
}

// ensureBytesTicker starts the ticker that resets the bytes-per-tick budget,
// the first time it is needed. It is a no-op on every call after the first,
// and, once Stop has won the race to run bytesOnce's function instead (see
// Stop), forever after: bytesTicker is then left nil, which callers must
// check for, see waitBytesPerTick.
func (c *Controller) ensureBytesTicker() {
	c.bytesOnce.Do(func() {
		c.bytesTicker = time.NewTicker(c.opts.bytesInterval)
	})
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
	c.ensureBytesTicker()
	if c.bytesTicker == nil {
		// Stop won the race to start the ticker (see ensureBytesTicker) and
		// so has already closed stopCh: there is nothing left to wait for.
		return context.Canceled
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
	case <-c.stopCh:
		return context.Canceled
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
		c.ensureReqsTicker()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return context.Canceled
		case <-c.reqsTokens:
		}
	}
	err := c.waitBytesPerTick(ctx)
	if err != nil && c.opts.reqsPerTick > 0 {
		select {
		case c.reqsTokens <- struct{}{}:
		default:
		}
	}
	return err
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
	c.stopOnce.Do(func() {
		close(c.stopCh)
		// If a ticker has not yet been lazily started by ensureReqsTicker /
		// ensureBytesTicker, running their Once here first wins the race and
		// permanently prevents either from ever starting: reqsTicker and
		// bytesTicker are then left nil forever, which Wait and
		// waitBytesPerTick already accommodate. If a ticker was already
		// started, or wins a concurrent race against this call instead, this
		// Do call is a no-op, and sync.Once's happens-before guarantee makes
		// it safe to read reqsTicker/bytesTicker below regardless of which
		// goroutine's function actually ran.
		c.reqsOnce.Do(func() {})
		c.bytesOnce.Do(func() {})
		if c.reqsTicker != nil {
			c.reqsTicker.Stop()
		}
		if c.bytesTicker != nil {
			c.bytesTicker.Stop()
		}
	})
}
