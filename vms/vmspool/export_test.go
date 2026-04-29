// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import "cloudeng.io/vms"

// InjectVM wraps inst in a vmsInstance with discard stdout/stderr and sends it
// into the pool's ready channel. Used in tests that need to fill the channel
// to capacity in order to provoke a blocking send from a replenishment goroutine.
func (p *Pool) InjectVM(inst vms.Instance) {
	p.ready <- &vmsInstance{Instance: inst, stdout: discardReadWriteCloser{}, stderr: discardReadWriteCloser{}}
}
