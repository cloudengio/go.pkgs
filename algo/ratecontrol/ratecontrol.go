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
// A single Controller paces every goroutine that shares it: the limits set by
// WithRequestsPerTick and WithBytesPerTick are aggregate limits and do not
// scale with the number of callers.
//
// Requests are paced by a token bucket, so the configured number is available
// as an immediate burst, is replenished over the tick interval, and is taken
// by callers in whatever order they happen to arrive.
//
// The bytes budget is refreshed once per tick interval. A caller proceeds
// immediately while that budget is unspent, but once it is exhausted callers
// queue and each tick admits whichever of them has been waiting longest, one
// per tick. No caller can therefore claim the budget again while another is
// still waiting: with n callers saturating the limiter, each is admitted every
// n ticks. Note that the budget gates admission only -- a caller reports what
// it transferred via BytesTransferred once it is through -- so an admitted
// request can still overshoot by however much it goes on to transfer.
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

	bytesMu sync.Mutex
	// bytesWaiters is the FIFO queue of callers parked waiting for budget,
	// longest-waiting first. Each tick admits the head, so admissions rotate
	// between callers rather than being won by whichever happens to wake first.
	bytesWaiters []chan struct{}

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
// along with the goroutine that admits a waiter on each tick, the first time
// either is needed. It is a no-op on every call after the first, and, once Stop
// has won the race to run bytesOnce's function instead (see Stop), forever
// after: bytesTicker is then left nil, which callers must check for, see
// waitBytesPerTick. It must be called with bytesMu held, so that a caller
// cannot enqueue itself before the goroutine that will admit it exists.
func (c *Controller) ensureBytesTicker() {
	c.bytesOnce.Do(func() {
		c.bytesTicker = time.NewTicker(c.opts.bytesInterval)
		go func() {
			for {
				select {
				case <-c.bytesTicker.C:
					c.refreshBudget()
				case <-c.stopCh:
					return
				}
			}
		}()
	})
}

// refreshBudget starts a new interval: it resets the budget and admits the
// caller at the head of the queue, ie. the one that has been waiting longest.
// Only one caller is admitted per tick because the budget gates admission
// before the size of a request is known: admitting more would risk overshooting
// the configured rate by however much they each go on to transfer.
func (c *Controller) refreshBudget() {
	c.bytesPerTick.Store(0)
	c.bytesMu.Lock()
	defer c.bytesMu.Unlock()
	if len(c.bytesWaiters) == 0 {
		return
	}
	admitted := c.bytesWaiters[0]
	c.bytesWaiters = c.bytesWaiters[1:]
	close(admitted)
}

// removeWaiter dequeues ready, for a caller that is abandoning its wait. It is
// a no-op if the caller was admitted concurrently, in which case that
// admission goes unused and the next tick admits the following waiter.
func (c *Controller) removeWaiter(ready chan struct{}) {
	c.bytesMu.Lock()
	defer c.bytesMu.Unlock()
	for i, w := range c.bytesWaiters {
		if w == ready {
			c.bytesWaiters = append(c.bytesWaiters[:i], c.bytesWaiters[i+1:]...)
			return
		}
	}
}

func (c *Controller) remaining(current *atomic.Int64, allowed int) bool {
	if allowed == 0 {
		return true
	}
	return current.Load() < int64(allowed)
}

// waitBytesPerTick blocks until the bytes-per-tick budget allows another
// request to proceed.
//
// The budget is enforced strictly and shared fairly across concurrent callers.
// A caller that finds the budget exhausted joins a FIFO queue and each tick
// admits the caller at its head, so the aggregate rate does not scale with the
// number of callers and no caller can claim the budget repeatedly while another
// waits: with n callers saturating the limiter, each is admitted every n ticks.
// Callers already queued take precedence over one arriving here, which is why
// the fast path below declines to proceed while the queue is occupied even when
// the budget would otherwise allow it.
//
// Note that the budget gates admission only: a caller reports what it
// transferred via BytesTransferred once it is through, so a single admitted
// request can still overshoot by however much it goes on to transfer. That is
// also why a tick admits one caller rather than several, since the size of a
// request is not known until after it has been admitted.
func (c *Controller) waitBytesPerTick(ctx context.Context) error {
	if c.opts.bytesPerTick == 0 {
		return nil
	}
	c.bytesMu.Lock()
	if len(c.bytesWaiters) == 0 && c.remaining(&c.bytesPerTick, c.opts.bytesPerTick) {
		c.bytesMu.Unlock()
		return nil
	}
	c.ensureBytesTicker()
	if c.bytesTicker == nil {
		// Stop won the race to start the ticker (see ensureBytesTicker) and
		// so has already closed stopCh: there is nothing left to wait for.
		c.bytesMu.Unlock()
		return context.Canceled
	}
	// Enqueue under the same lock as the check above, so that a tick cannot
	// slip between the two and leave this caller waiting out an interval whose
	// budget it should have been admitted against.
	ready := make(chan struct{})
	c.bytesWaiters = append(c.bytesWaiters, ready)
	c.bytesMu.Unlock()

	select {
	case <-ready:
		// Admitted by refreshBudget: this interval's budget is ours.
		return nil
	case <-ctx.Done():
		c.removeWaiter(ready)
		return ctx.Err()
	case <-c.stopCh:
		c.removeWaiter(ready)
		return context.Canceled
	}
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
