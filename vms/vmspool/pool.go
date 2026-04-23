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
	"log/slog"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/vms"
)

// Constructor is a function that creates a new, uninitialized VM instance.
// Each call must return a distinct instance.
type Constructor interface {
	New() vms.Instance
	Name() string
}

// Pool manages a fixed-size set of suspended virtual machine instances.
type Pool struct {
	options     options
	constructor Constructor
	ready       chan vms.Instance // suspended VMs waiting to be acquired
	done        chan struct{}     // closed by Close to signal pool shutdown
	closeOnce   sync.Once

	bgCtx  context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // tracks in-flight replenishment goroutines
}

type options struct {
	size   int
	logger *slog.Logger
}

const (
	DefaultPoolSize = 2
)

type Option func(*options)

// WithSize sets the number of VMs to maintain in the pool. The default is
// DefaultPoolSize.
func WithSize(size uint) Option {
	return func(o *options) {
		o.size = int(size)
	}
}

// WithLogger sets the logger used to report pool events and errors. The default is
// the logger from the context at the time of Pool creation.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// New returns a Pool that will maintain size suspended VMs using constructor.
// Call Start to fill the pool before calling Acquire.
func New(ctx context.Context, constructor Constructor, opts ...Option) *Pool {
	var options options
	options.size = DefaultPoolSize
	options.logger = ctxlog.Logger(ctx)
	for _, opt := range opts {
		opt(&options)
	}
	options.logger = options.logger.With("vmpool", constructor.Name(), "size", options.size)
	options.logger.Info("creating VM pool")
	return &Pool{
		options:     options,
		constructor: constructor,
		ready:       make(chan vms.Instance, options.size),
		done:        make(chan struct{}),
	}
}

// Start fills the pool with size suspended VMs. It blocks until all VMs are
// ready or any creation step fails. The context governs both the initial fill
// and background replenishment goroutines launched during the pool's lifetime.
func (p *Pool) Start(ctx context.Context) error {
	p.bgCtx, p.cancel = context.WithCancel(ctx)

	type result struct{ err error }
	results := make(chan result, p.options.size)
	for range p.options.size {
		go func() {
			results <- result{err: p.createVM(p.bgCtx)}
		}()
	}

	var errs errors.M
	for range p.options.size {
		errs.Append((<-results).err)
	}
	return errs.Err()
}

// createVM clones, starts, and suspends a new instance then places it in the
// ready channel. Returns an error if any step fails or the context is done.
// Any partially-created instance is cleaned up before returning an error.
func (p *Pool) createVM(ctx context.Context) error {
	bgCtx := context.Background()
	inst := p.constructor.New()
	if err := inst.Clone(ctx); err != nil {
		// Clone transitions from Initial; nothing to clean up beyond the
		// instance itself, which is already in Initial/Deleted state.
		return fmt.Errorf("vmspool: clone: %w", err)
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		// Instance is Stopped after Clone; clean it up.
		_ = vms.CleanupVM(bgCtx, inst)
		return fmt.Errorf("vmspool: start: %w", err)
	}
	if err := inst.Suspend(ctx); err != nil {
		// Instance may be Running; stop and delete it.
		_ = vms.CleanupVM(bgCtx, inst)
		return fmt.Errorf("vmspool: suspend: %w", err)
	}
	select {
	case p.ready <- inst:
		return nil
	case <-ctx.Done():
		_ = vms.CleanupVM(bgCtx, inst)
		return ctx.Err()
	}
}

// replenish creates one VM and adds it to the pool. Silently drops the VM if
// the pool is shutting down. Intended to run as a goroutine.
func (p *Pool) replenish() {
	defer p.wg.Done()
	_ = p.createVM(p.bgCtx)
}

// Acquire waits for a suspended VM, starts it, and returns a handle. The
// caller must call VM.Release when finished with the VM. Acquire blocks until
// a VM is available, ctx is cancelled, or the pool is closed.
func (p *Pool) Acquire(ctx context.Context) (*VM, error) {
	var inst vms.Instance
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.done:
		return nil, fmt.Errorf("vmspool: pool is closed")
	case inst = <-p.ready:
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		// Start failed; clean up the VM and replenish so the pool stays full.
		_ = vms.CleanupVM(context.Background(), inst)
		p.wg.Add(1)
		go p.replenish()
		return nil, fmt.Errorf("vmspool: acquire: %w", err)
	}
	return &VM{inst: inst, pool: p}, nil
}

// Close stops accepting new acquires, waits for all replenishment goroutines
// to finish, then deletes every suspended VM remaining in the pool.
func (p *Pool) Close(ctx context.Context) error {
	p.closeOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
		close(p.done)
	})
	p.wg.Wait()

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
	if err := vms.CleanupVM(ctx, v.inst); err != nil {
		return fmt.Errorf("vmspool: release: %w", err)
	}
	v.pool.wg.Add(1)
	go v.pool.replenish()
	return nil
}
