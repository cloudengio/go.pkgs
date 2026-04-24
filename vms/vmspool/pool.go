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

	// make start and close idempotent.
	startOnce sync.Once
	startErr  error
	closeOnce sync.Once

	bgCtx  context.Context
	cancel context.CancelFunc

	opMutex sync.Mutex // guards start, acquire and close operations

	// mu guards closed and serialises wg.Add with Close's wg.Wait, preventing
	// sync.WaitGroup misuse when Release/Acquire race with Close.
	mu     sync.Mutex
	closed bool
	wg     ctxsync.WaitGroup // tracks in-flight replenishment goroutines
}

type options struct {
	size             int
	statusCh         chan<- Event
	cleanupTimeout   time.Duration
	replenishTimeout time.Duration
}

const (
	DefaultPoolSize         = 2
	DefaultCleanupTimeout   = time.Minute
	DefaultReplenishTimeout = 5 * time.Minute
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

// WithReplenishTimeout sets the timeout for creating VMs during replenishment.
// The default is DefaultReplenishTimeout.
// A 0 or negative value is treated as DefaultReplenishTimeout.
func WithReplenishTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout <= 0 {
			timeout = DefaultReplenishTimeout
		}
		o.replenishTimeout = timeout
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

// New returns a Pool that will maintain size suspended VMs using constructor.
// Call Start to fill the pool before calling Acquire.
func New(constructor Constructor, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.cleanupTimeout = DefaultCleanupTimeout
	options.replenishTimeout = DefaultReplenishTimeout
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
// ready or any creation step fails. The context governs both the initial fill
// and background replenishment goroutines launched during the pool's lifetime.
func (p *Pool) Start(ctx context.Context) error {
	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	p.startOnce.Do(func() {
		p.startErr = p.start(ctx)
	})
	return p.startErr
}

func (p *Pool) start(ctx context.Context) error {
	p.bgCtx, p.cancel = context.WithCancel(ctx) //nolint:gosec // G118 false positive
	type result struct{ err error }
	results := make(chan result, p.options.size)
	for range p.options.size {
		go func() {
			results <- result{err: p.createVM()}
		}()
	}

	var errs errors.M
	for range p.options.size {
		errs.Append((<-results).err)
	}
	return errs.Err()
}

func (p *Pool) cleanupVM(inst vms.Instance) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), p.options.cleanupTimeout)
	_ = vms.CleanupVM(cleanupCtx, inst)
	cancel()
}

// createVM clones, starts, and suspends a new instance then places it in the
// ready channel. Returns an error if any step fails or the context is done.
// Any partially-created instance is cleaned up before returning an error.
func (p *Pool) createVM() error {
	// uses p.bgCtx so that cancellation from Close can interrupt
	// creation and clean up
	inst := p.constructor.New()
	if err := inst.Clone(p.bgCtx); err != nil {
		// Clone transitions from Initial; nothing to clean up beyond the
		// instance itself, which is already in Initial/Deleted state.
		return fmt.Errorf("vmspool: clone: %w", err)
	}
	if err := inst.Start(p.bgCtx, io.Discard, io.Discard); err != nil {
		// Instance is Stopped after Clone; clean it up.
		p.cleanupVM(inst)
		return fmt.Errorf("vmspool: start: %w", err)
	}
	if err := inst.Suspend(p.bgCtx); err != nil {
		// Instance may be Running; stop and delete it.
		p.cleanupVM(inst)
		return fmt.Errorf("vmspool: suspend: %w", err)
	}
	select {
	case p.ready <- inst:
		return nil
	case <-p.bgCtx.Done():
		p.cleanupVM(inst)
		return p.bgCtx.Err()
	}
}

// scheduleReplenish launches a replenishment goroutine unless the pool is
// already closed. wg.Add and the closed check are performed under mu so that
// Close cannot call wg.Wait between the check and the Add.
func (p *Pool) scheduleReplenish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.wg.Add(1)
	p.notify(EventReplenishStarted, nil)
	go p.replenish()
}

xxxxx - should replenish start over

// replenish creates one VM and adds it to the pool. Silently drops the VM if
// the pool is shutting down. Intended to run as a goroutine.
func (p *Pool) replenish() {
	defer p.wg.Done()
	if err := p.createVM(); err != nil {
		p.notify(EventReplenishFailed, err)
	} else {
		p.notify(EventReplenished, nil)
	}
}

// Acquire waits for a suspended VM, starts it, and returns a handle. The
// caller must call VM.Release when finished with the VM. Acquire blocks until
// a VM is available, ctx is cancelled, or the pool is closed.
func (p *Pool) Acquire(ctx context.Context, stdout, stderr io.Writer) (*VM, error) {
	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		err := fmt.Errorf("vmspool: pool is closed")
		p.notify(EventAttemptToUseClosedPool, err)
		return nil, err
	}
	p.notify(EventAcquireWaiting, nil)
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
	}
	p.notify(EventVMDequeued, nil)
	if err := inst.Start(ctx, stdout, stderr); err != nil {
		// Start failed; clean up the VM and replenish so the pool stays full.
		p.cleanupVM(inst)
		p.scheduleReplenish()
		err = fmt.Errorf("vmspool: acquire: %w", err)
		p.notify(EventAcquireFailed, err)
		return nil, err
	}
	p.notify(EventAcquired, nil)
	return &VM{inst: inst, pool: p}, nil
}

// Close stops accepting new acquires, waits for all replenishment goroutines
// to finish, then deletes every suspended VM remaining in the pool.
func (p *Pool) Close(ctx context.Context) error {
	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	p.closeOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
		close(p.done)
	})
	// Mark closed under mu before waiting, so no concurrent scheduleReplenish
	// can call wg.Add after wg.Wait begins.
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.wg.Wait(ctx)

	var errs errors.M
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
	v.pool.scheduleReplenish()
	v.pool.notify(EventReleased, nil)
	return nil
}
