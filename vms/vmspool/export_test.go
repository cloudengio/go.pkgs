// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import "cloudeng.io/vms"

// ReadyCh exposes the ready channel for use in tests that need to
// manipulate pool state directly (e.g. to fill the channel to capacity
// in order to provoke a blocking send from a replenishment goroutine).
func (p *Pool) ReadyCh() chan vms.Instance { return p.ready }
