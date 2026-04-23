// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:generate go run gen.go
package vms

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"cloudeng.io/errors"
)

// Properties represents the properties of a virtual machine instance.
type Properties struct {
	IP string // The IP address of the instance, if available.

	// Suspendable bool // Whether the instance supports being suspended.

	// SSHArgs     []string // SSH command line arguments to connect to the instance, e.g. user, host, port etc.
}

// Instance represents a virtual machine instance that can be managed through a
// lifecycle of states.
// Operations change the state of the instance as indicated below for
// successful operations. Error returning operations will either leave the
// state unchange, or transition to StateErrorUnknown if the state cannot
// be determined. Intermediate states (eg. Stopping, Starting) may
// be observed while the operation is in progress.
type Instance interface {

	// Clone prepares an instance for being stated. It should be
	// a synchronous operation and when it returns the state should be Stopped.
	// States: success: [Initial, Deleted] -> Cloning -> Stopped
	// States:   error: [Initial, Deleted] -> Cloning -> Initial
	Clone(ctx context.Context) error

	// Start starts the instance. It returns once the instance is running.
	// States: success: [Stopped] -> Starting -> Running
	// States:   error: [Stopped] -> Starting -> StateErrorUnknown or Stopped
	Start(ctx context.Context, stdout, stderr io.Writer) error

	// Stop stops the instance. It returns once the instance is stopped.
	// The timeout parameter specifies how long to wait for a graceful shutdown
	// before forcefully shutting down the vm instance.
	// States: success: [Running] -> Stopping -> Stopped; ; [Stopped] -> Stopped
	// States:   error: [Running] -> Stopping -> Stopped or StateErrorUnknown
	Stop(ctx context.Context, timeout time.Duration) (runErr, stopErr error)

	// Suspendable returns true if the instance supports being suspended.
	Suspendable() bool

	// Suspend suspends the instance. It returns once the instance is suspended.
	// States: success: [Running] -> Suspending -> Suspended; [Suspended]
	// States:   error: [Running] -> Suspending -> Suspended or StateErrorUnknown
	Suspend(ctx context.Context) error

	// Delete deletes the instance.
	// States: success: [Stopped, Suspended, ErrorUnknown] -> Deleting -> Deleted
	// States:   error: [Stopped, Suspended, ErrorUnknown] -> Deleting -> Deleted or StateErrorUnknown
	Delete(ctx context.Context) error

	// State returns the current state of the instance, it may be
	// called at any time.
	State(ctx context.Context) State

	// Exec executes the given command in the instance and returns when the
	// command completes.
	// Exec does not alter the state of the instance.
	Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error

	// Properties returns the properties of a running instance.
	// Properties does not alter the state of an instance.
	Properties(ctx context.Context) (Properties, error)
}

// State represents the state of a virtual machine instance.
type State int

const (
	StateInitial State = iota
	StateCloning
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateSuspending
	StateSuspended
	StateDeleting
	StateDeleted
	StateErrorUnknown
)

// Action represents an operation that causes a state transition.
type Action int

const (
	ActionNone Action = iota
	ActionClone
	ActionStart
	ActionStop
	ActionSuspend
	ActionDelete
)

// transitionTable maps (current state, action) to the next state.
// Pairs absent from the table are invalid transitions.
var transitionTable = map[State]map[Action]State{
	StateInitial: {
		ActionClone: StateCloning,
	},
	StateCloning: {
		ActionNone: StateCloning, // No-op to allow waiting for clone to complete
	},
	StateStarting: {
		ActionNone: StateStarting, // No-op to allow waiting for start to complete
	},
	StateRunning: {
		ActionStop:    StateStopping,
		ActionSuspend: StateSuspending,
	},
	StateStopping: {
		ActionNone: StateStopping, // No-op to allow waiting for stop to complete
	},
	StateStopped: {
		ActionStart:  StateStarting,
		ActionDelete: StateDeleting,
		ActionStop:   StateStopped, // make Stop idempotent.
	},
	StateSuspending: {
		ActionNone: StateSuspending, // No-op to allow waiting for suspend to complete
	},
	StateSuspended: {
		ActionStart:   StateStarting,
		ActionDelete:  StateDeleting,
		ActionSuspend: StateSuspended, // make Suspend idempotent
	},
	StateDeleting: {
		ActionNone: StateDeleting, // No-op to allow waiting for delete to complete
	},
	StateDeleted: {
		ActionClone: StateCloning, // Allow cloning from deleted state to create a new instance
	},
	StateErrorUnknown: {
		ActionDelete: StateDeleting, // Allow deleting from error state to clean up
	},
}

// Transition returns the next State reached by applying action to from,
// or false if the transition is not valid.
func (s State) Transition(action Action) (State, bool) {
	if actions, ok := transitionTable[s]; ok {
		if next, ok := actions[action]; ok {
			return next, true
		}
	}
	return s, false
}

// ValidActions returns the set of actions that are valid from the given state.
func (s State) ValidActions() []Action {
	actions := make([]Action, 0, len(transitionTable[s]))
	for a := range transitionTable[s] {
		actions = append(actions, a)
	}
	return actions
}

// Allowed returns true if the given action is valid from the current state.
func (s State) Allowed(action Action) bool {
	_, ok := s.Transition(action)
	return ok
}

func (s State) String() string {
	switch s {
	case StateInitial:
		return "Initial"
	case StateCloning:
		return "Cloning"
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateStopping:
		return "Stopping"
	case StateStopped:
		return "Stopped"
	case StateSuspending:
		return "Suspending"
	case StateSuspended:
		return "Suspended"
	case StateDeleting:
		return "Deleting"
	case StateDeleted:
		return "Deleted"
	case StateErrorUnknown:
		return "ErrorUnknown"
	default:
		return fmt.Sprintf("State(%d)", int(s))
	}
}

func (a Action) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionClone:
		return "Clone"
	case ActionStart:
		return "Start"
	case ActionStop:
		return "Stop"
	case ActionSuspend:
		return "Suspend"
	case ActionDelete:
		return "Delete"
	default:
		return fmt.Sprintf("Action(%d)", int(a))
	}
}

// PrintStates writes a human-readable description of every state and its
// valid transitions to out.
func PrintStates(out io.Writer) {
	// Collect and sort states for deterministic output.
	states := make([]State, 0, len(transitionTable))
	for s := range transitionTable {
		states = append(states, s)
	}
	slices.Sort(states)

	for _, s := range states {
		actions := transitionTable[s]
		// Sort actions for deterministic output.
		sorted := make([]Action, 0, len(actions))
		for a := range actions {
			sorted = append(sorted, a)
		}
		slices.Sort(sorted)

		transitions := make([]string, 0, len(sorted))
		for _, a := range sorted {
			next := actions[a]
			if a == ActionNone {
				transitions = append(transitions, fmt.Sprintf("(waiting) -> %s", next))
			} else {
				transitions = append(transitions, fmt.Sprintf("%s -> %s", a, next))
			}
		}
		fmt.Fprintf(out, "%-14s  %s\n", s, strings.Join(transitions, ",  "))
	}
}

// WaitForState polls inst.State until it returns the requested final
// state or the context is done. If intermediate states are provided, it also
// checks that any intermediate states returned by inst.State are in the set of
// allowed intermediate states on the way to the final state, returning an
// error if an unexpected intermediate state is observed.
func WaitForState(ctx context.Context, inst Instance, interval time.Duration, final State, intermediate ...State) error {
	if interval <= 0 {
		return fmt.Errorf("vms: WaitForState: interval must be positive: %v", interval)
	}
	found := func() (bool, error) {
		got := inst.State(ctx)
		if got == final {
			return true, nil
		}
		if len(intermediate) > 0 && !slices.Contains(intermediate, got) {
			return true, fmt.Errorf("unexpected intermediate state %s, want %v on the way to %s", got, intermediate, final)
		}
		return false, nil
	}

	if done, err := found(); done {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if done, err := found(); done {
				return err
			}
		}
	}
}

// CleanupVM attempts to clean up the given instance by stopping and deleting
// it if necessary. Suspended VMs are stopped before deletion.
// It returns an error if any of the operations fail.
func CleanupVM(ctx context.Context, inst Instance) error {
	s := inst.State(ctx)
	switch s {
	case StateDeleted, StateInitial:
		return nil
	case StateRunning:
		if runErr, stopErr := inst.Stop(ctx, time.Second*30); runErr != nil || stopErr != nil {
			var errs errors.M
			errs.Append(runErr)
			errs.Append(stopErr)
			return fmt.Errorf("cleanup: failed to stop VM: %w", errs.Err())
		}
		s = inst.State(ctx)
		if s != StateStopped {
			return fmt.Errorf("cleanup: expected VM to be stopped after stopping, got %s", s)
		}
		fallthrough
	case StateStopped, StateSuspended, StateErrorUnknown:
		if err := inst.Delete(ctx); err != nil {
			return fmt.Errorf("cleanup: failed to delete VM: %w", err)
		}
	default:
		return fmt.Errorf("cleanup: unexpected VM state %s", s)
	}
	return nil
}

var (
	ErrVMNotFound   = errors.New("VM not found")
	ErrVMNotRunning = errors.New("VM not running")
)
