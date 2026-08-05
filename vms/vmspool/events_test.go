// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"testing"

	"cloudeng.io/vms/vmspool"
)

// TestEventKindString guards against an EventKind being added, or inserted
// between existing ones, without a corresponding name.
func TestEventKindString(t *testing.T) {
	for e := vmspool.EventAcquireWaiting; e <= vmspool.EventAcquiredVMRetained; e++ {
		if got := e.String(); got == "" || got == "Unknown" {
			t.Errorf("EventKind(%d): missing name: got %q", int(e), got)
		}
	}
	for _, e := range []vmspool.EventKind{-1, vmspool.EventAcquiredVMRetained + 1} {
		if got, want := e.String(), "Unknown"; got != want {
			t.Errorf("EventKind(%d): got %v, want %v", int(e), got, want)
		}
	}
}
