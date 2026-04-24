// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package vmspool manages a fixed-size pool of suspended virtual machine
// instances. The pool pre-creates and suspends VMs so they can be started
// quickly when acquired. When a caller releases a VM it is deleted and a
// new one is created asynchronously to restore the pool to its target size.
package vmspool

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/ctxsync"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/vms"
)

// Constructor is an interface used to create new, uninitialized VM instances.
// Each call must return a distinct vms.Instance.
type Constructor interface {
	New() vms.Instance
}

// Pool manages a fixed-size set of suspended virtual machine instances.
type Pool struct {
	options     options
	constructor Constructor
	ready       chan vms.Instance // suspended VMs waiting to be acquired
	done        chan struct{}     // closed by Close to signal pool shutdown

	replenishCtx    context.Context
	replenishCancel context.CancelFunc

	opMutex sync.Mutex // guards start, acquire and close operations

	// mu guards closed and serialises wg.Add with Close's wg.Wait, preventing
	// sync.WaitGroup misuse when Release/Acquire race with Close.
	closedMu sync.Mutex
	closed   bool

	wg ctxsync.WaitGroup // tracks in-flight replenishment goroutines
}

type options struct {
	size              int
	statusCh          chan<- Event
	suspendVMs        bool
	cleanupTimeout    time.Duration
	replenishTimeout  time.Duration
	replenishInterval time.Duration
}

const (
	DefaultPoolSize       = 2
	DefaultCleanupTimeout = time.Minute
	DefaultCreateTimeout  = 5 * time.Minute
	DefaultCreateInterval = 500 * time.Millisecond
)

type Option func(*options)

// WithSize sets the number of VMs to maintain in the pool. The default is
// DefaultPoolSize. A 0 or negative value is treated as DefaultPoolSize.
func WithSize(size int) Option {
	return func(o *options) {
		if size <= 0 {
			size = DefaultPoolSize
		}
		o.size = size
	}
}

// WithCleanupTimeout sets the timeout for cleaning up VMs during Acquire and Close.
// The default is DefaultCleanupTimeout.
// A 0 or negative value is treated as DefaultCleanupTimeout.
func WithCleanupTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout <= 0 {
			timeout = DefaultCleanupTimeout
		}
		o.cleanupTimeout = timeout
	}
}

// WithCreateTimeoutAndInterval sets the timeout for creating a single
// VM and the interval between creation attempts.
// The default timeout and interval are DefaultCreateTimeout and DefaultCreateInterval.
// A 0 or negative value is treated as DefaultCreateTimeout or DefaultCreateInterval.
func WithCreateTimeoutAndInterval(timeout, interval time.Duration) Option {
	return func(o *options) {
		if timeout <= 0 {
			timeout = DefaultCreateTimeout
		}
		if interval <= 0 {
			interval = DefaultCreateInterval
		}
		o.replenishTimeout = timeout
		o.replenishInterval = interval
	}
}

// WithStatus registers ch to receive pool lifecycle events. Sends are
// non-blocking: events are dropped if ch is full. The caller is responsible
// for sizing the channel appropriately and draining it promptly.
func WithStatus(ch chan<- Event) Option {
	return func(o *options) {
		o.statusCh = ch
	}
}

// WithSuspendVMs configures the pool to suspend VMs after starting them during
// creation and replenishment. By default, VMs are suspended.
func WithSuspendVMs(suspend bool) Option {
	return func(o *options) {
		o.suspendVMs = suspend
	}
}

// New returns a Pool that will maintain size suspended VMs using constructor.
// Call Start to fill the pool before calling Acquire.
func New(constructor Constructor, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.cleanupTimeout = DefaultCleanupTimeout
	options.replenishTimeout = DefaultCreateTimeout
	options.replenishInterval = DefaultCreateInterval
	options.suspendVMs = true
	for _, opt := range opts {
		opt(&options)
	}
	return &Pool{
		options:     options,
		constructor: constructor,
		ready:       make(chan vms.Instance, options.size),
		done:        make(chan struct{}),
	}
}

func (p *Pool) notify(kind EventKind, err error) {
	if p.options.statusCh == nil {
		return
	}
	select {
	case p.options.statusCh <- Event{Time: time.Now(), Kind: kind, Err: err}:
	default:
	}
}

// Start fills the pool with size suspended VMs. It blocks until all VMs are
// ready or the context is canceled.
// Start can only be called once and will return an error if called more than once.
// After Start returns, the pool is ready to accept Acquire calls.
func (p *Pool) Start(ctx context.Context) error {
	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	if p.replenishCancel != nil {
		return fmt.Errorf("vmspool: pool already started")
	}
	p.replenishCtx = context.WithoutCancel(ctx) // detached context for replenishment goroutines;
	// p.replinishCancel must be called by close.
	p.replenishCtx, p.replenishCancel = context.WithCancel(p.replenishCtx)
	return p.fill(ctx, p.options.size)
}

func (p *Pool) fill(ctx context.Context, size int) error {
	var g errgroup.T
	for range size {
		g.GoContext(ctx, func() error {
			inst, err := p.createVMAndNotify(ctx)
			if err != nil {
				return err
			}
			select {
			case p.ready <- inst:
				return nil
			case <-ctx.Done():
				p.cleanupVM(inst)
				return ctx.Err()
			}
		})
	}
	return g.Wait()
}

func (p *Pool) cleanupVM(inst vms.Instance) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), p.options.cleanupTimeout)
	_ = vms.CleanupVM(cleanupCtx, inst)
	cancel()
}

// createVM clones, starts, and suspends a new instance then places it in the
// ready channel. Returns an error if any step fails or the context is done.
// Any partially-created instance is cleaned up before returning an error.
func (p *Pool) createVM(ctx context.Context) (vms.Instance, error) {
	inst := p.constructor.New()
	if err := inst.Clone(ctx); err != nil {
		// Clone transitions from Initial; nothing to clean up beyond the
		// instance itself, which is already in Initial/Deleted state.
		return nil, fmt.Errorf("vmspool: clone: %w", err)
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		// Instance is Stopped after Clone; clean it up.
		p.cleanupVM(inst)
		return nil, fmt.Errorf("vmspool: start: %w", err)
	}
	if !p.options.suspendVMs {
		return inst, nil
	}
	if err := inst.Suspend(ctx); err != nil {
		// Instance may be Running; stop and delete it.
		p.cleanupVM(inst)
		return nil, fmt.Errorf("vmspool: suspend: %w", err)
	}
	return inst, nil
}

func (p *Pool) createVMAndNotify(ctx context.Context) (vms.Instance, error) {
	p.notify(EventVMCreateStarted, nil)
	inst, err := p.createVM(ctx)
	if err != nil {
		p.notify(EventVMCreateFailed, err)
		return nil, err
	}
	p.notify(EventVMCreated, nil)
	return inst, nil
}

// requestReplenish launches a replenishment goroutine unless the pool is
// already closed. The closed check and wg.Add are performed under closedMu so
// that Close cannot call wg.Wait in the window between the check and the Add.
func (p *Pool) requestReplenish() {
	p.closedMu.Lock()
	if p.closed {
		p.closedMu.Unlock()
		return
	}
	p.wg.Go(func() {
		err := p.createVMLoop(p.replenishCtx, p.options.replenishInterval, p.options.replenishTimeout)
		if err != nil {
			// Log the error but keep the pool running; a later replenishment may succeed and restore capacity.
			p.notify(EventReplenishFailed, err)
			return
		}
		p.notify(EventReplenished, nil)
	})
	p.closedMu.Unlock()
	p.notify(EventReplenishStarted, nil)

}

// createVMLoop runs a loop that tries to create a new VM and add it to the pool
// until the pool is closed or the context is done.
func (p *Pool) createVMLoop(ctx context.Context, interval, timeout time.Duration) error {
	if p.attemptCreateVM(ctx, timeout) == nil {
		return nil
	}
	// Keep retrying to replenish the pool.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if p.attemptCreateVM(ctx, timeout) == nil {
				return nil
			}
		}
	}
}

// createVM creates a single VM and adds it to the pool.
func (p *Pool) attemptCreateVM(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	inst, err := p.createVMAndNotify(ctx)
	if err != nil {
		return err
	}
	select {
	case p.ready <- inst:
		return nil
	case <-ctx.Done():
		p.cleanupVM(inst)
		return ctx.Err()
	}
}

func (p *Pool) isClosed() bool {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()
	return p.closed
}

func (p *Pool) setClosed() bool {
	p.closedMu.Lock()
	defer p.closedMu.Unlock()
	closed := p.closed
	p.closed = true
	return closed
}

// Acquire waits for a suspended VM, starts it, and returns a handle. The
// caller must call VM.Release when finished with the VM. Acquire blocks until
// a VM is available, ctx is cancelled, or the pool is closed.
func (p *Pool) Acquire(ctx context.Context, stdout, stderr io.Writer) (*VM, error) {
	if p.isClosed() {
		err := fmt.Errorf("vmspool: pool is closed")
		p.notify(EventAttemptToUseClosedPool, err)
		return nil, err
	}
	p.notify(EventAcquireWaiting, nil)

	// Block without holding any lock so that Close can run concurrently and
	// signal shutdown by closing p.done.
	var inst vms.Instance
	select {
	case <-ctx.Done():
		p.notify(EventAcquireFailed, ctx.Err())
		return nil, ctx.Err()
	case <-p.done:
		err := fmt.Errorf("vmspool: pool is closed")
		p.notify(EventAttemptToUseClosedPool, err)
		return nil, err
	case inst = <-p.ready:
		if p.isClosed() {
			p.cleanupVM(inst)
			err := fmt.Errorf("vmspool: pool is closed")
			p.notify(EventAttemptToUseClosedPool, err)
			return nil, err
		}
	}
	p.notify(EventVMDequeued, nil)

	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	if p.options.suspendVMs {
		if err := inst.Start(ctx, stdout, stderr); err != nil {
			// Start failed; clean up the VM and replenish so the pool stays full.
			p.cleanupVM(inst)
			p.requestReplenish()
			err = fmt.Errorf("vmspool: acquire: %w", err)
			p.notify(EventAcquireFailed, err)
			return nil, err
		}
	}
	p.notify(EventAcquired, nil)
	return &VM{inst: inst, pool: p}, nil
}

// Close stops accepting new acquires, waits for all replenishment goroutines
// to finish, then deletes every suspended VM remaining in the pool. Close
// is idempotent.
func (p *Pool) Close(ctx context.Context) error {
	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	if closed := p.setClosed(); closed { // already closed
		return nil
	}
	if p.replenishCancel != nil {
		p.replenishCancel() // signal replenishment goroutines to stop
	}
	close(p.done) // signal pool shutdown to unblock Acquire calls
	p.wg.Wait(ctx)

	// capture error but continue to cleanup VMs.
	var errs errors.M
	if err := ctx.Err(); err != nil {
		errs.Append(err)
	}
	for {
		select {
		case inst := <-p.ready:
			errs.Append(vms.CleanupVM(ctx, inst))
		default:
			return errs.Err()
		}
	}
}

// VM is a running virtual machine instance acquired from a Pool.
// Use Exec to run commands and Release when done.
type VM struct {
	inst vms.Instance
	pool *Pool
}

// Exec runs cmd with args inside the VM, writing output to stdout and stderr.
func (v *VM) Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error {
	return v.inst.Exec(ctx, stdout, stderr, cmd, args...)
}

// Release deletes the VM and asynchronously replenishes the pool with a new
// suspended instance. It must be called exactly once per acquired VM.
func (v *VM) Release(ctx context.Context) error {
	v.pool.notify(EventRelease, nil)
	if err := vms.CleanupVM(ctx, v.inst); err != nil {
		return fmt.Errorf("vmspool: release: %w", err)
	}
	v.pool.requestReplenish()
	v.pool.notify(EventReleased, nil)
	return nil
}
