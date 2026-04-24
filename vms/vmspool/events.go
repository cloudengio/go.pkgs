// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import "time"

// EventKind identifies the type of pool event sent to a status channel.
type EventKind int

const (
	// EventAcquireWaiting is emitted when Acquire is called and blocks
	// waiting for a suspended VM to become available.
	EventAcquireWaiting EventKind = iota

	// EventVMDequeued is emitted when a suspended VM is taken from the pool
	// and is about to be started for the caller.
	EventVMDequeued

	// EventAcquired is emitted when the VM has been started and is returned
	// to the caller.
	EventAcquired

	// EventAcquireFailed is emitted when Acquire returns an error (context
	// cancelled or VM start failure). Err is set.
	EventAcquireFailed

	// EventAttemptToUseClosedPool is emitted when Acquire is called on a pool
	// that is already closed or has been signalled to close. Err is set.
	EventAttemptToUseClosedPool

	// EventRelease is emitted when Release is called by the caller.
	EventRelease

	// EventReleased is emitted after the VM has been deleted and
	// replenishment has been scheduled.
	EventReleased

	// EventCreateStarted is emitted when a goroutine is launched to create a new VM
	// to place in the pool.
	EventVMCreateStarted

	// EventVMCreated is emitted when a new VM has been successfully created.
	EventVMCreated

	// EventVMCreateFailed is emitted when VM creation fails.
	EventVMCreateFailed

	// EventReplenishStarted is emitted when a replenishment goroutine is
	// launched to restore the pool to its target size.
	EventReplenishStarted

	// EventReplenished is emitted when a new VM has been suspended and
	// placed in the pool, restoring one unit of capacity.
	EventReplenished

	// EventReplenishFailed is emitted when VM creation during replenishment
	// fails. The pool shrinks by one until a later replenishment succeeds.
	// Err is set.
	EventReplenishFailed
)

func (e EventKind) String() string {
	switch e {
	case EventAcquireWaiting:
		return "AcquireWaiting"
	case EventVMDequeued:
		return "VMDequeued"
	case EventAcquired:
		return "Acquired"
	case EventAcquireFailed:
		return "AcquireFailed"
	case EventAttemptToUseClosedPool:
		return "AttemptToUseClosedPool"
	case EventRelease:
		return "Release"
	case EventReleased:
		return "Released"
	case EventVMCreateStarted:
		return "VMCreateStarted"
	case EventVMCreated:
		return "VMCreated"
	case EventVMCreateFailed:
		return "VMCreateFailed"
	case EventReplenishStarted:
		return "ReplenishStarted"
	case EventReplenished:
		return "Replenished"
	case EventReplenishFailed:
		return "ReplenishFailed"
	default:
		return "Unknown"
	}
}

// Event describes a single pool lifecycle event.
type Event struct {
	Time time.Time
	Kind EventKind
	Err  error // non-nil for *Failed events
}
