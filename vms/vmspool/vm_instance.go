// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/vms"
)

type vmsInstance struct {
	vms.Instance
	stdout, stderr io.Writer
	// acquired records that a caller holds the VM. It is written under
	// Pool.opMutex, but is atomic so that Pool.Stats can read it without
	// taking that lock, which Acquire holds while starting a VM and Close holds
	// for the whole of its shutdown.
	acquired atomic.Bool

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
// a second replacement would grow the pool beyond its configured size. For the
// same reason only the first call to Delete replenishes the pool: any further
// call finds the VM already deleted and does nothing.
func (v *VM) Delete(ctx context.Context) error {
	v.pool.notify(EventRelease, nil)
	claimed, cleanupErr := v.pool.claimAndCleanupVM(ctx, v.inst, v.pool.options.stopTimeout)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("vmspool: delete: %w", cleanupErr)
	}
	// Only replenish if this call claimed the VM AND atomically transitioned
	// stopped to true. If StopAndRelease already stopped it, setStopped(true)
	// returns false. If Delete is called twice, claimed is false.
	if claimed && v.inst != nil && v.inst.setStopped(true) {
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

// PoolName returns the name of the pool to which the VM belongs.
func (v *VM) PoolName() string {
	if v.pool == nil {
		return ""
	}
	return v.pool.name
}

// Provider returns the provider used to create and manage the VM.
func (v *VM) Provider() Provider {
	if v.pool == nil {
		return nil
	}
	return v.pool.provider
}
