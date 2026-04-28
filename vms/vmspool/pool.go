// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package vmspool manages a fixed-size pool of suspended or stopped virtual
// machine instances. The pool pre-creates and suspends VMs so they can be
// started quickly when acquired. When a caller releases a VM it is deleted and a
// new one is created asynchronously to restore the pool to its target size.
// Note that if the underlying VM implementation does not support suspend/resume,
// the pool will create the VM and leave it stopped until acquired.
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

type vmsInstance struct {
	vms.Instance
	stdout, stderr io.ReadWriteCloser
	stopped        bool
}

// Pool manages a fixed-size set of suspended virtual machine instances.
type Pool struct {
	options     options
	constructor Constructor
	ready       chan vmsInstance // suspended VMs waiting to be acquired
	done        chan struct{}    // closed by Close to signal pool shutdown

	opMutex sync.Mutex // guards acquire and close operations

	// mu guards closed, replenishCtx, replenishCancel
	// and serialises wg.Add with Close's wg.Wait, preventing
	// sync.WaitGroup misuse when Release/Acquire race with Close.
	mu              sync.Mutex
	closed          bool
	replenishCtx    context.Context
	replenishCancel context.CancelFunc
	// tracks in-flight replenishment and vm creationg goroutines
	wg ctxsync.WaitGroup
}

type options struct {
	size             int
	statusCh         chan<- Event
	stagingBehaviour StagingBehaviour
	cleanupTimeout   time.Duration
	createTimeout    time.Duration
	createInterval   time.Duration
	stopTimeout      time.Duration
	createStdout     func(id string) (io.ReadWriteCloser, error)
	createStderr     func(id string) (io.ReadWriteCloser, error)
}

const (
	DefaultPoolSize       = 2
	DefaultCleanupTimeout = time.Minute
	DefaultCreateTimeout  = 5 * time.Minute
	DefaultCreateInterval = 500 * time.Millisecond
	DefaultStopTimeout    = time.Minute
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
		o.createTimeout = timeout
		o.createInterval = interval
	}
}

// WithStopTimeout sets the timeout for stopping VMs.
// The default is DefaultStopTimeout.
// A 0 or negative value is treated as DefaultStopTimeout.
func WithStopTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout <= 0 {
			timeout = DefaultStopTimeout
		}
		o.stopTimeout = timeout
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

// WithStagingBehaviour sets the staging behaviour for VMs in the pool. The default is StagingBehaviourRunning.
func WithStagingBehaviour(behaviour StagingBehaviour) Option {
	return func(o *options) {
		o.stagingBehaviour = behaviour
	}
}

// StagingBehaviour determines the state of VMs in the pool after creation but
// before acquisition. The default is StagingBehaviourRunning. The behaviours are:
//   - StagingBehaviourRunning: VMs are left running and Acquire will hand them to the caller as-is.
//   - StagingBehaviourSuspended: VMs are suspended and Acquire will resume them before handing them to the caller provided that the VM supports suspend/resume; if not, the pool falls back to StagingBehaviourStopped behaviour.
//   - StagingBehaviourStopped: VMs are stopped and Acquire will start them before handing them to the caller.
type StagingBehaviour int

func (s StagingBehaviour) String() string {
	switch s {
	case StagingBehaviourSuspended:
		return "Suspended"
	case StagingBehaviourRunning:
		return "Running"
	case StagingBehaviourStopped:
		return "Stopped"
	}
	return "Unknown"
}

const (
	StagingBehaviourRunning StagingBehaviour = iota
	StagingBehaviourSuspended
	StagingBehaviourStopped
)

type discardReadWriteCloser struct{}

func (discardReadWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (discardReadWriteCloser) Close() error {
	return nil
}

func (discardReadWriteCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

// WithStdoutStderr configures the pool to use the provided functions to create
// stdout and stderr pipes for VMs during creation and replenishment. The
// value of vms.Instance.ID() is passed to the stdout function and can be used to create
// uniquely identifiable pipes. If either function is nil, a no-op ReadWriteCloser is used
// that discards all writes and returns EOF on reads.
func WithStdoutStderr(stdout, stderr func(id string) (io.ReadWriteCloser, error)) Option {
	return func(o *options) {
		if stdout == nil {
			stdout = func(string) (io.ReadWriteCloser, error) {
				return discardReadWriteCloser{}, nil
			}
		}
		if stderr == nil {
			stderr = func(string) (io.ReadWriteCloser, error) {
				return discardReadWriteCloser{}, nil
			}
		}
		o.createStdout = stdout
		o.createStderr = stderr
	}
}

// New returns a Pool that will maintain size suspended VMs using constructor.
// Call Start to fill the pool before calling Acquire.
func New(constructor Constructor, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.cleanupTimeout = DefaultCleanupTimeout
	options.createTimeout = DefaultCreateTimeout
	options.createInterval = DefaultCreateInterval
	options.stopTimeout = DefaultStopTimeout
	options.stagingBehaviour = StagingBehaviourRunning
	options.createStdout = func(string) (io.ReadWriteCloser, error) { return discardReadWriteCloser{}, nil }
	options.createStderr = func(string) (io.ReadWriteCloser, error) { return discardReadWriteCloser{}, nil }
	for _, opt := range opts {
		opt(&options)
	}
	return &Pool{
		options:     options,
		constructor: constructor,
		ready:       make(chan vmsInstance, options.size),
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

// Start blocks until at least one VM is ready to be acquired (or the context is
// canceled), any other VMs required to fill the pool are created asynchronously.
// Start can be called once only and will return an error if called more than once.
// After Start returns, the pool is ready to accept Acquire calls.
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.replenishCancel != nil {
		p.mu.Unlock()
		return fmt.Errorf("vmspool: pool already started")
	}
	p.mu.Unlock()
	p.replenishCtx = context.WithoutCancel(ctx) // detached context for replenishment goroutines;
	// p.replenishCancel must be called by Close.
	p.replenishCtx, p.replenishCancel = context.WithCancel(p.replenishCtx)
	return p.fill(ctx, p.options.size)
}

func (p *Pool) fill(ctx context.Context, size int) error {
	err := p.createVMWithRetry(ctx, p.options.createInterval, p.options.createTimeout)
	if err != nil {
		return err
	}

	// at least one VM is ready; launch goroutine to fill the rest of the pool so
	// Start can return and the pool can be used immediately.
	p.wg.Go(func() {
		var g errgroup.T
		for range size - 1 {
			g.GoContext(p.replenishCtx, func() error {
				return p.createVMWithRetry(p.replenishCtx, p.options.createInterval, p.options.createTimeout)
			})
		}
		if g.Wait() == nil {
			p.notify(EventStartPoolFull, nil)
		}
	})
	return nil
}

func (p *Pool) cleanupVM(inst vmsInstance) {
	if inst.Instance == nil {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), p.options.cleanupTimeout)
	_ = vms.CleanupVM(cleanupCtx, inst.Instance, p.options.stopTimeout)
	if inst.stdout != nil {
		_ = inst.stdout.Close()
	}
	if inst.stderr != nil {
		_ = inst.stderr.Close()
	}
	cancel()
}

// createVM clones, starts, and suspends a new instance then places it in the
// ready channel. Returns an error if any step fails or the context is done.
// Any partially-created instance is cleaned up before returning an error.
func (p *Pool) createVM(ctx context.Context) (vmsInstance, error) {
	inst := p.constructor.New()
	if inst == nil {
		return vmsInstance{}, fmt.Errorf("vmspool: constructor returned nil instance")
	}

	if err := inst.Clone(ctx); err != nil {
		// Clone transitions from Initial; nothing to clean up beyond the
		// instance itself, which is already in Initial/Deleted state.
		return vmsInstance{}, fmt.Errorf("vmspool: clone: %w", err)
	}

	stdout, err := p.options.createStdout(inst.ID())
	if err != nil {
		return vmsInstance{}, fmt.Errorf("vmspool: create stdout: %w", err)
	}
	stderr, err := p.options.createStderr(inst.ID())
	if err != nil {
		stdout.Close()
		return vmsInstance{}, fmt.Errorf("vmspool: create stderr: %w", err)
	}
	vmsInst := vmsInstance{Instance: inst, stdout: stdout, stderr: stderr}

	// leave VM in stopped state.
	if p.options.stagingBehaviour == StagingBehaviourStopped ||
		(!inst.Suspendable() && (p.options.stagingBehaviour == StagingBehaviourSuspended)) {
		vmsInst.stopped = true
		return vmsInst, nil
	}

	if err := vmsInst.Start(ctx, stdout, stderr); err != nil {
		// Instance is Stopped after Clone; clean it up.
		p.cleanupVM(vmsInst)
		return vmsInstance{}, fmt.Errorf("vmspool: start: %w", err)
	}

	if p.options.stagingBehaviour == StagingBehaviourRunning || !inst.Suspendable() {
		return vmsInst, nil
	}

	vmsInst.stopped = true
	if err := inst.Suspend(ctx); err != nil {
		// Instance may be Running; stop and delete it.
		p.cleanupVM(vmsInst)
		return vmsInstance{}, fmt.Errorf("vmspool: suspend: %w", err)
	}
	return vmsInst, nil
}

func (p *Pool) createVMAndNotify(ctx context.Context) (vmsInstance, error) {
	p.notify(EventVMCreateStarted, nil)
	inst, err := p.createVM(ctx)
	if err != nil {
		p.notify(EventVMCreateFailed, err)
		return vmsInstance{}, err
	}
	p.notify(EventVMCreated, nil)
	return inst, nil
}

// requestReplenish launches a replenishment goroutine unless the pool is
// already closed. The closed check and wg.Go are performed under mu so
// that Close cannot call wg.Wait in the window between the check and the Add.
func (p *Pool) requestReplenish() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.notify(EventReplenishStarted, nil)
	p.wg.Go(func() {
		err := p.createVMWithRetry(p.replenishCtx, p.options.createInterval, p.options.createTimeout)
		if err != nil {
			// Log the error but keep the pool running; a later replenishment may succeed and restore capacity.
			p.notify(EventReplenishFailed, err)
			return
		}
		p.notify(EventReplenished, nil)
	})
	p.mu.Unlock()

}

// createVMWithRetry runs a loop that tries to create a new VM and add it to the pool
// until the pool is closed or the context is done.
func (p *Pool) createVMWithRetry(ctx context.Context, interval, timeout time.Duration) error {
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

// attemptCreateVM creates a single VM and adds it to the pool.
func (p *Pool) attemptCreateVM(ctx context.Context, timeout time.Duration) error {
	var mu sync.Mutex
	var inst vmsInstance
	var err error
	doneCh := make(chan struct{})

	p.wg.Go(func() {
		mu.Lock()
		defer mu.Unlock()
		inst, err = p.createVMAndNotify(ctx)
		close(doneCh)
	})

	select {
	case <-doneCh:
		if err != nil {
			p.cleanupVM(inst)
			return err
		}
	case <-ctx.Done():
		mu.Lock()
		defer mu.Unlock()
		p.cleanupVM(inst)
		return ctx.Err()
	case <-time.After(timeout):
		mu.Lock()
		defer mu.Unlock()
		p.cleanupVM(inst)
		return fmt.Errorf("vmspool: create VM timed out after %s", timeout)
	}

	// this is racy since if the context is canceled and the select may
	// unblock due to either ctx.Done or p.done; if ctx.Done is selected,
	// the created VM is cleaned up immediately, but if p.done is selected,
	// the VM is added to the pool and will be cleaned up later by Close.
	select {
	case p.ready <- inst:
		return nil
	case <-ctx.Done():
		p.cleanupVM(inst)
		return ctx.Err()
	}
}

func (p *Pool) isClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

func (p *Pool) setClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	closed := p.closed
	p.closed = true
	return closed
}

// Acquire waits for a suspended VM, starts it, and returns a handle. The
// caller must call VM.Release when finished with the VM. Acquire blocks until
// a VM is available, ctx is cancelled, or the pool is closed.
func (p *Pool) Acquire(ctx context.Context) (*VM, error) {
	if p.isClosed() {
		err := fmt.Errorf("vmspool: pool is closed")
		p.notify(EventAttemptToUseClosedPool, err)
		return nil, err
	}
	p.notify(EventAcquireWaiting, nil)

	// Block without holding any lock so that Close can run concurrently and
	// signal shutdown by closing p.done.
	var inst vmsInstance
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
	if inst.stopped {
		if err := inst.Start(ctx, inst.stdout, inst.stderr); err != nil {
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
			cleanupCtx, cancel := context.WithTimeout(context.Background(), p.options.cleanupTimeout)
			errs.Append(vms.CleanupVM(cleanupCtx, inst, p.options.stopTimeout))
			if inst.stdout != nil {
				errs.Append(inst.stdout.Close())
			}
			if inst.stderr != nil {
				errs.Append(inst.stderr.Close())
			}
			cancel()
		default:
			return errs.Err()
		}
	}
}

// VM is a running virtual machine instance acquired from a Pool.
// Use Exec to run commands and Release when done.
type VM struct {
	inst vmsInstance
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
	cleanupErr := vms.CleanupVM(ctx, v.inst, v.pool.options.stopTimeout)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("vmspool: release: %w", cleanupErr)
	}
	var errs errors.M
	errs.Append(cleanupErr)
	if v.inst.stdout != nil {
		errs.Append(v.inst.stdout.Close())
	}
	if v.inst.stderr != nil {
		errs.Append(v.inst.stderr.Close())
	}
	v.pool.requestReplenish()
	v.pool.notify(EventReleased, nil)
	return errs.Err()
}
