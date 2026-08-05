// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"io"

	"cloudeng.io/vms"
)

// InjectVM wraps inst in a vmsInstance with discard stdout/stderr and sends it
// into the pool's ready channel. Used in tests that need to fill the channel
// to capacity in order to provoke a blocking send from a replenishment goroutine.
// The instance is tracked, preserving the invariant that every instance in the
// ready channel is claimable for cleanup.
func (p *Pool) InjectVM(inst vms.Instance) {
	vmsInst := &vmsInstance{Instance: inst, stdout: io.Discard, stderr: io.Discard}
	p.track(vmsInst)
	p.ready <- vmsInst
}

// NewTestVM builds a VM backed by inst for use by external tests, which cannot
// construct the unexported vmsInstance directly. Pass a nil inst to exercise the
// nil-underlying-instance guard.
func NewTestVM(inst vms.Instance) *VM {
	return &VM{inst: &vmsInstance{Instance: inst}}
}

// Stopped reports the VM's internal stopped flag, set by Stop on success.
func (v *VM) Stopped() bool {
	return v.inst != nil && v.inst.stopped
}
