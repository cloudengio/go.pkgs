// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package vmspool manages a fixed-size pool of suspended or stopped virtual
// machine instances. The pool pre-creates and mantains VMs according to the
// requested StagingBehaviour so they can be  started quickly when acquired.
// An acquired VM is deleted by VM.Delete, and a new one is created
// asynchronously to restore the pool to its target size. A caller that calls
// VM.StopAndRelease first triggers that replacement at the point of the stop
// rather than at the delete, since a stopped VM will not run again.
package vmspool

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/ctxsync"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/vms"
)

// VMInfo is a backend-neutral summary of a VM managed by a Provider. It holds
// only the lightweight fields that are cheap to obtain in bulk via
// Provider.List.
type VMInfo struct {
	Name     string
	Pool     string
	State    string // backend-specific state string, e.g. "running", "stopped"
	Running  bool
	Accessed time.Time // best-effort last activity time; may be creation time if last access is unavailable
}

// VMDetail extends VMInfo with the fuller, potentially more expensive per-VM
// details returned by Provider.Get, such as the resources allocated to the VM.
type VMDetail struct {
	VMInfo
	DiskGiB  int // size of the VM's disk in GiB
	NumCores int // number of CPU cores allocated to the VM
	MemGiB   int // amount of RAM allocated to the VM in GiB
}

// Provider creates and manages the VMs for a Pool. In addition to constructing
// new instances it can enumerate, inspect and delete the VMs it has created,
// which pools and cleanup tooling use for status reporting and reclaiming
// orphaned VMs.
type Provider interface {
	// New returns a new, uninitialized VM instance. Each call must return a
	// distinct vms.Instance. ctx governs any work done to construct the
	// instance. It returns an error if the instance could not be created.
	New(ctx context.Context) (vms.Instance, error)
	// List returns lightweight summaries of the VMs currently present for this
	// provider's pool.
	List(ctx context.Context) ([]VMInfo, error)
	// Get returns the full details of a single VM by name.
	Get(ctx context.Context, vmName string) (VMDetail, error)
	// Delete stops (if running) and deletes every VM belonging to this
	// provider's pool, returning the names deleted. It continues past individual
	// failures.
	Delete(ctx context.Context, stopTimeout time.Duration) ([]string, error)
}

type vmsInstance struct {
	vms.Instance
	stdout, stderr io.Writer
	acquired       bool // guarded by Pool.opMutex

	mu sync.Mutex
	// stopped records that the VM is not running: either because it is staged
	// stopped or suspended in the pool, or because the caller that acquired it
	// has stopped it. It determines whether Acquire must start the VM before
	// handing it over, and which of VM.StopAndRelease and VM.Delete requests
	// the replacement VM. GUARDED by mu.
	stopped bool
}

// setStopped records whether the VM is running and reports whether this call
// changed the flag. Only the caller that wins the transition to stopped
// requests replenishment, so repeated calls to VM.StopAndRelease cannot grow
// the pool beyond its configured size.
func (inst *vmsInstance) setStopped(stopped bool) bool {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	changed := inst.stopped != stopped
	inst.stopped = stopped
	return changed
}

func (inst *vmsInstance) isStopped() bool {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.stopped
}

// Pool manages a fixed-size set of suspended virtual machine instances.
type Pool struct {
	options  options
	provider Provider
	ready    chan *vmsInstance // suspended VMs waiting to be acquired
	done     chan struct{}     // closed by Close to signal pool shutdown

	opMutex sync.Mutex // guards acquire and close operations

	// mu guards closed, live, replenishCtx, replenishCancel
	// and serialises wg.Add with Close's wg.Wait, preventing
	// sync.WaitGroup misuse when Delete/Acquire race with Close.
	mu     sync.Mutex
	closed bool
	// live tracks every instance the pool has created and not yet deleted,
	// including those still being created, those queued in ready and those
	// acquired by a caller. Close uses it to delete VMs that never reached
	// ready, or that have left it, and which would otherwise be leaked.
	live            map[*vmsInstance]struct{}
	replenishCtx    context.Context
	replenishCancel context.CancelFunc
	// tracks in-flight replenishment and vm creation goroutines
	wg ctxsync.WaitGroup
}

type options struct {
	size                  int
	statusCh              chan<- Event
	stagingBehaviour      StagingBehaviour
	cleanupTimeout        time.Duration
	createTimeout         time.Duration
	createInterval        time.Duration
	stopTimeout           time.Duration
	createStdout          func(id string) io.Writer
	createStderr          func(id string) io.Writer
	deleteAcquiredOnClose bool
}

const (
	DefaultPoolSize       = 2
	DefaultCleanupTimeout = time.Minute
	DefaultCreateTimeout  = 5 * time.Minute
	DefaultCreateInterval = 500 * time.Millisecond
	DefaultStopTimeout    = time.Minute
)

type Config struct {
	Size                  int              `yaml:"size" doc:"The number of VMs to maintain in the pool. A 0 or negative value is treated as DefaultPoolSize."`
	CleanupTimeout        time.Duration    `yaml:"cleanup_timeout" doc:"The timeout for cleaning up VMs during Acquire and Close. A 0 or negative value is treated as DefaultCleanupTimeout."`
	CreateTimeout         time.Duration    `yaml:"create_timeout" doc:"The timeout for creating a single VM. A 0 or negative value is treated as DefaultCreateTimeout."`
	CreateInterval        time.Duration    `yaml:"create_interval" doc:"The interval between VM creation attempts. A 0 or negative value is treated as DefaultCreateInterval."`
	StopTimeout           time.Duration    `yaml:"stop_timeout" doc:"The timeout for stopping VMs. A 0 or negative value is treated as DefaultStopTimeout."`
	StagingBehaviour      StagingBehaviour `yaml:"staging_behaviour" doc:"The staging behaviour for VMs in the pool. The default is StagingBehaviourRunning. The behaviours are: StagingBehaviourRunning: VMs are left running and Acquire will hand them to the caller as-is. StagingBehaviourSuspended: VMs are suspended and Acquire will resume them before handing them to the caller provided that the VM supports suspend/resume; if not, the pool falls back to StagingBehaviourStopped behaviour. StagingBehaviourStopped: VMs are stopped and Acquire will start them before handing them to the caller."`
	DeleteAcquiredOnClose bool             `yaml:"delete_acquired_on_close" doc:"Controls whether Close deletes VMs that are still held by a caller, ie. that have been acquired but not yet deleted, whether or not the caller has stopped them with VM.StopAndRelease. The default is true, on the basis that the pool owns every VM it creates and Close is the last chance to delete them. Set it to false when callers outlive the pool and are responsible for calling VM.Delete themselves; Close then emits EventAcquiredVMRetained for each VM it leaves behind."`
}

// Options returns a slice of Option values derived from the Config fields.
// Zero or negative durations and sizes are left to the individual With* functions
// to replace with their documented defaults. It does not include WithStatus or
// WithStdoutStderr, which require non-serialisable values (channels, functions).
func (c Config) Options() []Option {
	return []Option{
		WithSize(c.Size),
		WithCleanupTimeout(c.CleanupTimeout),
		WithCreateTimeoutAndInterval(c.CreateTimeout, c.CreateInterval),
		WithStopTimeout(c.StopTimeout),
		WithStagingBehaviour(c.StagingBehaviour),
		WithDeleteAcquiredOnClose(c.DeleteAcquiredOnClose),
	}
}

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

// WithDeleteAcquiredOnClose controls whether Close deletes VMs that are still
// held by a caller, ie. that have been acquired but not yet deleted, whether or
// not the caller has stopped them with VM.StopAndRelease. The default is true,
// on the basis that the pool owns every VM it creates and Close is the last
// chance to delete them. Set it to false when callers outlive the pool and are
// responsible for calling VM.Delete themselves; Close then emits
// EventAcquiredVMRetained for each VM it leaves behind.
func WithDeleteAcquiredOnClose(v bool) Option {
	return func(o *options) {
		o.deleteAcquiredOnClose = v
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

// MarshalText implements encoding.TextMarshaler, emitting the string name of
// the behaviour. yaml.v3, encoding/json, and other text-based encoders will
// call this automatically.
func (s StagingBehaviour) MarshalText() ([]byte, error) {
	switch s {
	case StagingBehaviourRunning, StagingBehaviourSuspended, StagingBehaviourStopped:
		return []byte(s.String()), nil
	default:
		return nil, fmt.Errorf("vmspool: unknown staging behaviour %d", s)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler, accepting the string name
// of the behaviour case-insensitively ("Running", "Suspended", "Stopped").
// yaml.v3 calls this for string-valued YAML nodes, so no direct yaml import is
// needed in this package.
func (s *StagingBehaviour) UnmarshalText(b []byte) error {
	v := strings.TrimSpace(string(b))
	switch strings.ToLower(v) {
	case "running":
		*s = StagingBehaviourRunning
	case "suspended":
		*s = StagingBehaviourSuspended
	case "stopped":
		*s = StagingBehaviourStopped
	default:
		return fmt.Errorf("vmspool: unknown StagingBehaviour %q; valid values: Running, Suspended, Stopped", v)
	}
	return nil
}

const (
	StagingBehaviourRunning StagingBehaviour = iota
	StagingBehaviourSuspended
	StagingBehaviourStopped
)

// WithStdoutStderr configures the pool to use the provided functions to create
// stdout and stderr io.Writers for VMs during creation and replenishment. The
// value of vms.Instance.ID() is passed to the stdout function and can be used to create
// uniquely identifiable pipes. If either function is nil, a no-op Writer is used
// that discards all writes.
func WithStdoutStderr(stdout, stderr func(id string) io.Writer) Option {
	return func(o *options) {
		if stdout == nil {
			stdout = func(string) io.Writer {
				return io.Discard
			}
		}
		if stderr == nil {
			stderr = func(string) io.Writer {
				return io.Discard
			}
		}
		o.createStdout = stdout
		o.createStderr = stderr
	}
}

// New returns a Pool that will maintain size suspended VMs using provider.
// Call Start to fill the pool before calling Acquire.
func New(provider Provider, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.cleanupTimeout = DefaultCleanupTimeout
	options.createTimeout = DefaultCreateTimeout
	options.createInterval = DefaultCreateInterval
	options.stopTimeout = DefaultStopTimeout
	options.stagingBehaviour = StagingBehaviourRunning
	options.deleteAcquiredOnClose = true
	options.createStdout = func(string) io.Writer {
		return io.Discard
	}
	options.createStderr = func(string) io.Writer {
		return io.Discard
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &Pool{
		options:  options,
		provider: provider,
		ready:    make(chan *vmsInstance, options.size),
		done:     make(chan struct{}),
		live:     map[*vmsInstance]struct{}{},
	}
}

// track records inst as owned by the pool. Every instance returned by the
// constructor must be tracked before any operation is attempted on it so that
// Close can delete it however the creation turns out.
func (p *Pool) track(inst *vmsInstance) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.live[inst] = struct{}{}
}

// claim removes inst from the live set and reports whether the caller won
// ownership of its cleanup. Several paths may try to delete the same instance
// (a caller's Delete, Close's sweep of acquired VMs, the creation error
// paths); exactly one of them wins the claim and performs the cleanup, the
// rest treat it as already handled. In particular this is what makes Delete
// a no-op for a VM that Close has already deleted.
func (p *Pool) claim(inst *vmsInstance) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.live[inst]; !ok {
		return false
	}
	delete(p.live, inst)
	return true
}

// tracked returns the instances the pool owns that have not been deleted.
func (p *Pool) tracked() []*vmsInstance {
	p.mu.Lock()
	defer p.mu.Unlock()
	insts := make([]*vmsInstance, 0, len(p.live))
	for inst := range p.live {
		insts = append(insts, inst)
	}
	return insts
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

// cleanupVM stops and deletes inst provided the caller wins the claim for it;
// if another path has already claimed the instance the cleanup is theirs and
// cleanupVM returns nil immediately. A claimed instance is no longer tracked
// even if its cleanup fails: the error is returned to the initiating caller,
// and an external sweep (such as deleting VMs by name at startup) is the
// backstop for any VM leaked this way.
func (p *Pool) cleanupVM(ctx context.Context, inst *vmsInstance, timeout time.Duration) error {
	if inst == nil || inst.Instance == nil {
		return nil
	}
	if !p.claim(inst) {
		return nil
	}
	return vms.CleanupVM(ctx, inst.Instance, timeout)
}

// cleanupVMOnError is cleanupVM for paths that are already returning an error.
// It runs on a background context since the context that governs those paths
// has typically been cancelled, which is what brought them here.
func (p *Pool) cleanupVMOnError(inst *vmsInstance, timeout time.Duration) {
	_ = p.cleanupVM(context.Background(), inst, timeout)
}

// createVM clones, starts, and suspends a new instance then places it in the
// ready channel. Returns an error if any step fails or the context is done.
// Any partially-created instance is cleaned up before returning an error.
func (p *Pool) createVM(ctx context.Context) (*vmsInstance, error) {
	inst, err := p.provider.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("vmspool: provider failed to create instance: %w", err)
	}
	if inst == nil {
		return nil, fmt.Errorf("vmspool: provider returned nil instance")
	}

	// Track the instance before cloning it: a clone that fails, or that is
	// interrupted, may still have left a VM behind that only Close can delete.
	vmsInst := &vmsInstance{Instance: inst}
	p.track(vmsInst)

	if err := inst.Clone(ctx); err != nil {
		p.cleanupVMOnError(vmsInst, p.options.cleanupTimeout)
		return nil, fmt.Errorf("vmspool: clone: %w", err)
	}

	// Create the output writers only once there is a VM to write about;
	// creating them earlier would leave a trail of empty files behind for
	// every clone that fails.
	stdout := p.options.createStdout(inst.ID())
	stderr := p.options.createStderr(inst.ID())
	vmsInst.stdout, vmsInst.stderr = stdout, stderr

	// leave VM in stopped state.
	if p.options.stagingBehaviour == StagingBehaviourStopped ||
		(!inst.Suspendable() && (p.options.stagingBehaviour == StagingBehaviourSuspended)) {
		vmsInst.setStopped(true)
		return p.discardIfClosed(vmsInst)
	}

	if err := vmsInst.Start(ctx, stdout, stderr); err != nil {
		// Instance is Stopped after Clone; clean it up.
		p.cleanupVMOnError(vmsInst, p.options.cleanupTimeout)
		return nil, fmt.Errorf("vmspool: start: %w", err)
	}

	if p.options.stagingBehaviour == StagingBehaviourRunning || !inst.Suspendable() {
		return p.discardIfClosed(vmsInst)
	}

	vmsInst.setStopped(true)
	if err := inst.Suspend(ctx); err != nil {
		// Instance may be Running; stop and delete it.
		p.cleanupVMOnError(vmsInst, p.options.cleanupTimeout)
		return nil, fmt.Errorf("vmspool: suspend: %w", err)
	}
	return p.discardIfClosed(vmsInst)
}

// discardIfClosed deletes a newly created VM if the pool was closed while it
// was being created, since nothing will ever take it out of the ready channel.
func (p *Pool) discardIfClosed(inst *vmsInstance) (*vmsInstance, error) {
	if !p.isClosed() {
		return inst, nil
	}
	p.cleanupVMOnError(inst, p.options.cleanupTimeout)
	return nil, fmt.Errorf("vmspool: pool was closed while creating a VM")
}

func (p *Pool) createVMAndNotify(ctx context.Context) (*vmsInstance, error) {
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
	var inst *vmsInstance
	var err error
	doneCh := make(chan struct{})

	p.wg.Go(func() {
		inst, err = p.createVMAndNotify(ctx)
		close(doneCh)
	})

	// The abandoned cases below, where creation is still in progress when this
	// call gives up on it, do not clean up: reading inst would race with the
	// goroutine still writing it. The instance was tracked before its first
	// operation, so Close deletes whatever the goroutine ends up creating.
	select {
	case <-doneCh:
		if err != nil {
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
			return err
		}
	case <-ctx.Done():
		select {
		case <-doneCh:
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
		default:
		}
		return ctx.Err()
	case <-time.After(timeout):
		select {
		case <-doneCh:
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
		default:
		}
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
		p.cleanupVMOnError(inst, p.options.cleanupTimeout)
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
// caller must call VM.Delete when finished with the VM. Acquire blocks until
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
	var inst *vmsInstance
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
			var errs errors.M
			errs.Append(p.cleanupVM(context.Background(), inst, p.options.cleanupTimeout))
			err := fmt.Errorf("vmspool: pool is closed")
			errs.Append(err)
			p.notify(EventAttemptToUseClosedPool, err)
			return nil, errs.Err()
		}
	}
	p.notify(EventVMDequeued, nil)

	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	if inst.isStopped() {
		if err := inst.Start(ctx, inst.stdout, inst.stderr); err != nil {
			// Start failed; clean up the VM and replenish so the pool stays full.
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
			p.requestReplenish()
			err = fmt.Errorf("vmspool: acquire: %w", err)
			p.notify(EventAcquireFailed, err)
			return nil, err
		}
		// The VM is running again, so it is the caller's StopAndRelease or Delete that
		// requests its replacement, not its staging state.
		inst.setStopped(false)
	}
	inst.acquired = true
	p.notify(EventAcquired, nil)
	return &VM{inst: inst, pool: p}, nil
}

// Close stops accepting new acquires, waits for all replenishment goroutines
// to finish, then deletes every VM the pool created whose deletion has not
// already been performed, or claimed, by another path (such as a concurrent
// Delete). That includes VMs queued in the pool, VMs abandoned part way
// through creation, and, unless WithDeleteAcquiredOnClose(false) was used, VMs
// that a caller acquired and has not deleted. Close is idempotent.
//
// A cancelled ctx bounds only the wait for the in-flight creation goroutines:
// Close then attempts to delete their VMs while the creation operations are
// still running. Such attempts fail with 'unexpected VM state' errors that are
// included in the returned error, and those VMs are instead deleted by the
// creation goroutines' own cleanup once they observe the cancellation or the
// closed pool. Deletion itself is not bounded by ctx.
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
	// Drain the VMs waiting to be acquired.
drained:
	for {
		select {
		case inst := <-p.ready:
			errs.Append(p.cleanupVM(context.Background(), inst, p.options.stopTimeout))
		default:
			break drained
		}
	}
	// Delete whatever is left: VMs abandoned during creation and VMs still
	// held by a caller. Deleting them concurrently keeps Close bounded by a
	// single VM's stop timeout rather than one per VM.
	var g errgroup.T
	for _, inst := range p.tracked() {
		if inst.acquired && !p.options.deleteAcquiredOnClose {
			p.notify(EventAcquiredVMRetained, nil)
			continue
		}
		g.Go(func() error {
			if !p.claim(inst) {
				// Another path (eg. a concurrent Delete) claimed the
				// instance after the tracked snapshot was taken; its
				// cleanup is theirs.
				return nil
			}
			if err := vms.CleanupVM(context.Background(), inst.Instance, p.options.stopTimeout); err != nil {
				// Return the claim so that the instance's owner can still
				// delete it via its own cleanup path; this is how a VM whose
				// creation outlived a Close bounded by a cancelled ctx gets
				// deleted rather than leaked.
				p.track(inst)
				return err
			}
			p.notify(EventOrphanedVMDeleted, nil)
			return nil
		})
	}
	errs.Append(g.Wait())
	return errs.Err()
}

// VM is a running virtual machine instance acquired from a Pool.
// Use Exec to run commands and Delete when done.
type VM struct {
	inst *vmsInstance
	pool *Pool
}

// Exec runs cmd with args inside the VM, writing output to stdout and stderr.
func (v *VM) Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error {
	if v.inst == nil || v.inst.Instance == nil {
		return fmt.Errorf("vmspool: invalid VM instance")
	}
	return v.inst.Exec(ctx, stdout, stderr, cmd, args...)
}

// StopAndRelease stops the VM and releases its slot in the pool, returning any
// error from the last command run in the VM and any error from stopping it.
// It is idempotent.
//
// A stopped VM is never handed out again, so the first successful stop
// asynchronously replenishes the pool rather than leaving the slot idle: the
// replacement is created while the caller is still finishing with the stopped
// VM (collecting logs, say). The VM itself is not deleted; the caller must
// still call Delete when done with it, which will not request a second
// replacement.
func (v *VM) StopAndRelease(ctx context.Context, timeout time.Duration) (runErr, stopErr error) {
	if v.inst == nil || v.inst.Instance == nil {
		return nil, fmt.Errorf("vmspool: invalid VM instance")
	}
	runErr, stopErr = v.inst.Stop(ctx, timeout)
	if stopErr == nil && v.inst.setStopped(true) {
		v.pool.requestReplenish()
	}
	return runErr, stopErr
}

// Delete deletes the VM, stopping it first if it is still running. It must be
// called exactly once per acquired VM. If the pool has been closed and deletes
// acquired VMs on close (the default), the VM has already been deleted by Close
// and Delete does not delete it again.
//
// Delete asynchronously replenishes the pool unless the VM has already been
// stopped by StopAndRelease, which released the slot at that point; requesting
// a second replacement would grow the pool beyond its configured size.
func (v *VM) Delete(ctx context.Context) error {
	v.pool.notify(EventRelease, nil)
	replenish := v.inst != nil && !v.inst.isStopped()
	cleanupErr := v.pool.cleanupVM(ctx, v.inst, v.pool.options.stopTimeout)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("vmspool: delete: %w", cleanupErr)
	}
	if replenish {
		v.pool.requestReplenish()
	}
	v.pool.notify(EventReleased, nil)
	return cleanupErr
}

// ID returns the unique identifier of the VM instance. It may be empty if the VM is invalid.
func (v *VM) ID() string {
	if v.inst == nil || v.inst.Instance == nil {
		return ""
	}
	return v.inst.ID()
}
