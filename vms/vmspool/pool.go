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

	"cloudeng.io/algo/ratecontrol"
	"cloudeng.io/errors"
	"cloudeng.io/os/executil"
	"cloudeng.io/sync/ctxsync"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/vms"
)

// Pool manages a fixed-size set of suspended virtual machine instances.
type Pool struct {
	name     string
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
	createBackoff         ratecontrol.ExponentialBackoffConfig
	stopTimeout           time.Duration
	createStdout          func(id string) io.Writer
	createStderr          func(id string) io.Writer
	deleteAcquiredOnClose bool
	completion            *CompletionQueue
}

const (
	DefaultPoolSize       = 2
	DefaultCleanupTimeout = time.Minute
	DefaultStopTimeout    = time.Minute
)

// DefaultCreateBackoff returns the default backoff used when creating VMs:
// a 500ms initial delay doubling over 10 steps, for a total delay budget of
// ~8.5 minutes, which is also the timeout allowed for a single creation
// attempt.
func DefaultCreateBackoff() ratecontrol.ExponentialBackoffConfig {
	return ratecontrol.ExponentialBackoffConfig{InitialDelay: 500 * time.Millisecond, Steps: 10}
}

type Config struct {
	Size                  int                                  `yaml:"size" doc:"The number of VMs to maintain in the pool. A 0 or negative value is treated as DefaultPoolSize."`
	CleanupTimeout        time.Duration                        `yaml:"cleanup_timeout" doc:"The timeout for cleaning up VMs during Acquire and Close. A 0 or negative value is treated as DefaultCleanupTimeout."`
	CreateBackoff         ratecontrol.ExponentialBackoffConfig `yaml:"create_backoff" doc:"The backoff applied between attempts to create a VM. Its total delay budget also bounds a single creation attempt. A configuration with a non-positive initial delay or number of steps is treated as DefaultCreateBackoff."`
	StopTimeout           time.Duration                        `yaml:"stop_timeout" doc:"The timeout for stopping VMs. A 0 or negative value is treated as DefaultStopTimeout."`
	StagingBehaviour      StagingBehaviour                     `yaml:"staging_behaviour" doc:"The staging behaviour for VMs in the pool. The default is StagingBehaviourRunning. The behaviours are: StagingBehaviourRunning: VMs are left running and Acquire will hand them to the caller as-is. StagingBehaviourSuspended: VMs are suspended and Acquire will resume them before handing them to the caller provided that the VM supports suspend/resume; if not, the pool falls back to StagingBehaviourStopped behaviour. StagingBehaviourStopped: VMs are stopped and Acquire will start them before handing them to the caller."`
	DeleteAcquiredOnClose bool                                 `yaml:"delete_acquired_on_close" doc:"Controls whether Close deletes VMs that are still held by a caller, ie. that have been acquired but not yet deleted, whether or not the caller has stopped them with VM.StopAndRelease. The default is true, on the basis that the pool owns every VM it creates and Close is the last chance to delete them. Set it to false when callers outlive the pool and are responsible for calling VM.Delete themselves; Close then emits EventAcquiredVMRetained for each VM it leaves behind."`
}

// Options returns a slice of Option values derived from the Config fields.
// Zero or negative durations and sizes are left to the individual With* functions
// to replace with their documented defaults. It does not include WithStatus or
// WithStdoutStderr, which require non-serialisable values (channels, functions).
func (c Config) Options() []Option {
	return []Option{
		WithSize(c.Size),
		WithCleanupTimeout(c.CleanupTimeout),
		WithCreateBackoff(c.CreateBackoff),
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

// WithCreateBackoff sets the backoff applied between attempts to create a VM.
// Its total delay budget also bounds how long a single creation attempt may
// take before it is abandoned and retried.
// The default is DefaultCreateBackoff(); a configuration with a non-positive
// initial delay or number of steps is replaced by it, so that an unset
// configuration cannot produce a NoBackoff that retries without pausing.
func WithCreateBackoff(cfg ratecontrol.ExponentialBackoffConfig) Option {
	return func(o *options) {
		if cfg.InitialDelay <= 0 || cfg.Steps <= 0 {
			cfg = DefaultCreateBackoff()
		}
		o.createBackoff = cfg
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

// WithCompletionQueue attaches a completion queue to the pool, which the VMs it
// hands out report through VM.CompletionQueue. It is how work running on a VM
// finds the queue to park that VM in when it is done, without the queue being
// plumbed through alongside the VM, and how an application reports on the VMs
// of this pool that the queue is still holding, see
// CompletionQueue.NumEventsPerPool.
//
// A single queue may be attached to any number of pools. The pool neither
// pushes to the queue nor closes it: parking finished work is the caller's
// decision, and the queue, which outlives any one pool that shares it, is
// closed by whoever created it.
func WithCompletionQueue(q *CompletionQueue) Option {
	return func(o *options) {
		o.completion = q
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

// New returns a Pool, named name, that will maintain size suspended VMs using
// provider. The name identifies the pool to an application that runs more than
// one and is reported by the VMs it hands out, see VM.PoolName; it is not used
// to name the VMs themselves, which the provider names.
// Call Start to fill the pool before calling Acquire.
func New(provider Provider, name string, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.cleanupTimeout = DefaultCleanupTimeout
	options.createBackoff = DefaultCreateBackoff()
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
		name:     name,
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

// Stats is a snapshot of the state of a Pool, see Pool.Stats.
type Stats struct {
	// Available is the number of VMs waiting in the pool that Acquire can
	// return without waiting for one to be created.
	Available int
	// Acquired is the number of VMs held by callers that have not yet deleted
	// them. A VM stopped by VM.StopAndRelease is still counted, since the
	// caller has yet to call VM.Delete, even though its slot in the pool has
	// already been given up and is being replenished.
	Acquired int
	// Pending is the number of VMs that exist but are neither available nor
	// acquired: those being created, whether by Start or to replenish the
	// pool, and those in transit between the pool and a caller.
	Pending int
}

// String implements fmt.Stringer.
func (s Stats) String() string {
	return fmt.Sprintf("available=%d acquired=%d pending=%d",
		s.Available, s.Acquired, s.Pending)
}

// Stats returns a snapshot of the pool's size and how many of its VMs are
// available, acquired and pending. It does not wait for any operation in
// progress, such as a VM being started by Acquire, and is safe to call from any
// goroutine at any time, including before Start and after Close.
//
// The counts are read together but the pool keeps changing while they are, so
// they describe a moment that may already have passed; a VM being handed over
// by Acquire, say, is counted as Pending until the caller holds it. The counts
// are consistent with each other, however: none is ever negative, and Available
// + Acquired + Pending is the number of VMs the pool has created and not yet
// deleted, which can exceed Size while replacements for stopped VMs are created.
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	acquired := 0
	for inst := range p.live {
		if inst.acquired.Load() {
			acquired++
		}
	}
	// A VM in ready is also in live, but Close may drain and claim VMs from
	// ready while a caller is between reading it and here, so clamp rather
	// than let the difference go negative.
	available := 0
	if !p.closed {
		available = min(len(p.ready), len(p.live)-acquired)
	}
	return Stats{
		Available: available,
		Acquired:  acquired,
		Pending:   len(p.live) - acquired - available,
	}
}

// Size returns the number of VMs the pool is configured to hold, see WithSize.
// It is fixed when the pool is created, so it is not part of Stats.
func (p *Pool) Size() int {
	return p.options.size
}

// Name returns the name the pool was created with. All VMS from this pool return
// this value via their PoolName method.
func (p *Pool) Name() string {
	return p.name
}

// CompletionQueue returns the queue that finished work from this pool is parked
// in, or nil if the pool has none, see WithCompletionQueue. All VMs from this
// pool return this value via their CompletionQueue method.
func (p *Pool) CompletionQueue() *CompletionQueue {
	return p.options.completion
}

// notify reports an event about the pool itself.
func (p *Pool) notify(kind EventKind, err error) {
	p.send(Event{Time: time.Now(), Kind: kind, Pool: p.name, Err: err})
}

// notifyVM reports an event about a single VM, naming it so that a consumer
// can attribute the event without parsing the text of an error, and so that
// the events of a pool's VMs can be followed individually.
func (p *Pool) notifyVM(kind EventKind, inst *vmsInstance, err error) {
	var id string
	if inst != nil && inst.Instance != nil {
		id = inst.ID()
	}
	p.send(Event{Time: time.Now(), Kind: kind, Pool: p.name, VM: id, Err: err})
}

// send delivers an event to the status channel, if one was registered. The
// send is non-blocking: an event is dropped rather than delaying the pool,
// see WithStatus.
func (p *Pool) send(e Event) {
	if p.options.statusCh == nil {
		return
	}
	select {
	case p.options.statusCh <- e:
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
	err := p.createVMWithRetry(ctx)
	if err != nil {
		return err
	}

	// at least one VM is ready; launch goroutine to fill the rest of the pool so
	// Start can return and the pool can be used immediately.
	p.wg.Go(func() {
		var g errgroup.T
		for range size - 1 {
			g.GoContext(p.replenishCtx, func() error {
				return p.createVMWithRetry(p.replenishCtx)
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
	_, err := p.claimAndCleanupVM(ctx, inst, timeout)
	return err
}

// claimAndCleanupVM is cleanupVM that also reports whether the caller won the
// claim, and so gave up the instance's slot in the pool, as opposed to finding
// that another path had already done so.
func (p *Pool) claimAndCleanupVM(ctx context.Context, inst *vmsInstance, timeout time.Duration) (claimed bool, err error) {
	if inst == nil || inst.Instance == nil {
		return false, nil
	}
	if !p.claim(inst) {
		return false, nil
	}
	return true, vms.CleanupVM(ctx, inst.Instance, timeout)
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
	p.notifyVM(EventVMCreated, inst, nil)
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
		err := p.createVMWithRetry(p.replenishCtx)
		if err != nil {
			// Log the error but keep the pool running; a later replenishment may succeed and restore capacity.
			p.notify(EventReplenishFailed, err)
			return
		}
		p.notify(EventReplenished, nil)
	})
	p.mu.Unlock()

}

// createVMWithRetry tries to create a new VM and add it to the pool, retrying
// with the configured backoff until it succeeds, the backoff is exhausted, or
// the context is done. The backoff's total delay budget is also the timeout for
// each individual attempt.
func (p *Pool) createVMWithRetry(ctx context.Context) error {
	timeout := p.options.createBackoff.TotalTimeout()
	var lastErr error
	err := executil.WaitForBackoff(ctx, p.options.createBackoff.NewBackoff(),
		func(ctx context.Context) (bool, error) {
			lastErr = p.attemptCreateVM(ctx, timeout)
			return lastErr == nil, nil
		})
	if err != nil && lastErr != nil {
		// Report why creation kept failing, not just that the retries ran out.
		return fmt.Errorf("%w: last attempt: %v", err, lastErr)
	}
	return err
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

	timer := time.NewTimer(timeout)
	defer timer.Stop()

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
	case <-timer.C:
		select {
		case <-doneCh:
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
		default:
		}
		return fmt.Errorf("vmspool: create VM timed out after %s", timeout)
	}

	// This select is racy if the context is cancelled concurrently: if
	// ctx.Done is selected, the created VM is cleaned up immediately; if
	// sending to ready is selected, the VM is queued in the pool and will be
	// cleaned up on Acquire or Close.
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
			p.notifyVM(EventAttemptToUseClosedPool, inst, err)
			return nil, errs.Err()
		}
	}
	p.notifyVM(EventVMDequeued, inst, nil)

	p.opMutex.Lock()
	defer p.opMutex.Unlock()
	if inst.isStopped() {
		if err := inst.Start(ctx, inst.stdout, inst.stderr); err != nil {
			// Start failed; clean up the VM and replenish so the pool stays full.
			p.cleanupVMOnError(inst, p.options.cleanupTimeout)
			p.requestReplenish()
			err = fmt.Errorf("vmspool: acquire: %w", err)
			p.notifyVM(EventAcquireFailed, inst, err)
			return nil, err
		}
		// The VM is running again, so it is the caller's StopAndRelease or Delete that
		// requests its replacement, not its staging state.
		inst.setStopped(false)
	}
	inst.acquired.Store(true)
	p.notifyVM(EventAcquired, inst, nil)
	return &VM{inst: inst, pool: p}, nil
}

// Refresh takes the VMs that are currently available out of the pool and
// deletes them, leaving the pool to create replacements, which is how a pool
// picks up a new base image without being restarted. It returns the number of
// VMs removed.
//
// VMs that are held by a caller are left alone: they are in use, and the work
// using them did not ask for a different image. Nor does Refresh wait for the
// replacements, which take as long as creating any VM does, or for the
// deletions themselves: it returns as soon as the VMs it is replacing can no
// longer be handed out. A VM that cannot be deleted is reported through the
// status channel as EventRefreshFailed.
//
// Refresh returns 0 for a pool with nothing available to replace, including a
// closed one.
func (p *Pool) Refresh(ctx context.Context) int {
	if ctx.Err() != nil || p.isClosed() {
		return 0
	}
	stats := p.Stats()
	if stats.Available == 0 {
		return 0
	}
	// Draining available VMs directly from ready takes them out of circulation
	// without starting stopped or suspended VMs, and without competing with
	// other callers waiting on ready.
	toRefresh := make([]*vmsInstance, 0, stats.Available)
drained:
	for range stats.Available {
		select {
		case inst := <-p.ready:
			toRefresh = append(toRefresh, inst)
		default:
			break drained
		}
	}
	if len(toRefresh) == 0 {
		return 0
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		for _, inst := range toRefresh {
			_ = p.cleanupVM(context.Background(), inst, p.options.stopTimeout)
		}
		return 0
	}
	p.wg.Go(func() {
		p.deleteAllInstances(toRefresh)
	})
	p.mu.Unlock()
	return len(toRefresh)
}

// deleteAllInstances deletes the VMs taken out of the pool by Refresh, which
// also replenishes the pool for each of them. Deletion is detached from the
// context that asked for the refresh: the VMs have already been taken out of
// circulation, and leaving one that has been undeleted would strand it.
// Deletions run with cleanupTimeout to prevent a hung provider from leaking
// goroutines indefinitely.
func (p *Pool) deleteAllInstances(instances []*vmsInstance) {
	var wg sync.WaitGroup
	for _, inst := range instances {
		wg.Go(func() {
			delCtx, cancel := context.WithTimeout(context.Background(), p.options.cleanupTimeout)
			defer cancel()
			claimed, err := p.claimAndCleanupVM(delCtx, inst, p.options.stopTimeout)
			if claimed {
				p.requestReplenish()
			}
			if err != nil {
				p.notifyVM(EventRefreshFailed, inst, fmt.Errorf("vmspool: refresh: delete %v: %w", inst.ID(), err))
			}
		})
	}
	wg.Wait()
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
		if inst.acquired.Load() && !p.options.deleteAcquiredOnClose {
			p.notifyVM(EventAcquiredVMRetained, inst, nil)
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
			p.notifyVM(EventOrphanedVMDeleted, inst, nil)
			return nil
		})
	}
	errs.Append(g.Wait())

	return errs.Err()
}
