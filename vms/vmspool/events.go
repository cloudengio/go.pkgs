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

	// EventVMDequeued is emitted when a suspended or running VM is taken from
	// the pool and is about to be started by the caller, or if running
	// returned as-is to the caller.
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

	// EventRelease is emitted when VM.Delete is called by the caller.
	EventRelease

	// EventReleased is emitted after the VM has been deleted and
	// replenishment has been scheduled.
	EventReleased

	// EventVMCreateStarted is emitted when a goroutine is launched to create a new VM
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

	// EventStartPoolFull is emitted when the asynchronous process to
	// fill the pool initiated by Start is completed.
	EventStartPoolFull

	// EventOrphanedVMDeleted is emitted by Close for each VM it deletes that
	// was not waiting in the pool: one abandoned part way through creation, or
	// one still held by a caller that never deleted it.
	EventOrphanedVMDeleted

	// EventAcquiredVMRetained is emitted by Close for each acquired VM it
	// leaves in place because WithDeleteAcquiredOnClose(false) was set. The
	// caller that holds the VM is responsible for deleting it.
	EventAcquiredVMRetained
)

// eventKindNames maps each EventKind to its name. It is indexed by the
// EventKind itself, so every EventKind must have an entry here.
var eventKindNames = [...]string{
	EventAcquireWaiting:         "AcquireWaiting",
	EventVMDequeued:             "VMDequeued",
	EventAcquired:               "Acquired",
	EventAcquireFailed:          "AcquireFailed",
	EventAttemptToUseClosedPool: "AttemptToUseClosedPool",
	EventRelease:                "Release",
	EventReleased:               "Released",
	EventVMCreateStarted:        "VMCreateStarted",
	EventVMCreated:              "VMCreated",
	EventVMCreateFailed:         "VMCreateFailed",
	EventReplenishStarted:       "ReplenishStarted",
	EventReplenished:            "Replenished",
	EventReplenishFailed:        "ReplenishFailed",
	EventStartPoolFull:          "StartPoolFull",
	EventOrphanedVMDeleted:      "OrphanedVMDeleted",
	EventAcquiredVMRetained:     "AcquiredVMRetained",
}

func (e EventKind) String() string {
	if uint(e) >= uint(len(eventKindNames)) {
		return "Unknown"
	}
	return eventKindNames[e]
}

// Event describes a single pool lifecycle event.
type Event struct {
	Time time.Time
	Kind EventKind
	Err  error // non-nil for *Failed events
}
