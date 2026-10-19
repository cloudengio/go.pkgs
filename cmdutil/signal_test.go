// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"cloudeng.io/cmdutil"
	cerrors "cloudeng.io/errors"
)

var errClosed = errors.New("closed")

// ExampleIsCleanShutdown illustrates the decision a CLI makes at its top level
// once its context has been cancelled, and the difference the cancelIsEnough
// flag makes to a command that aggregates errors from several components.
func ExampleIsCleanShutdown() {
	outcome := func(err error) string {
		if cmdutil.IsCleanShutdown(err, false) {
			return "clean"
		}
		return "failed"
	}

	fmt.Println("nil:", outcome(nil))
	fmt.Println("interrupt:", outcome(cmdutil.ErrInterrupt))
	fmt.Println("cancelled:", outcome(context.Canceled))
	fmt.Println("wrapped cancellation:", outcome(fmt.Errorf("serving: %w", context.Canceled)))
	fmt.Println("deadline exceeded:", outcome(context.DeadlineExceeded))
	fmt.Println("failure:", outcome(errors.New("disk full")))

	// A command that aggregates the errors from concurrent components reports
	// the cancellation alongside any genuine failure. Requiring every error to
	// be clean keeps that failure visible; accepting any cancellation hides it.
	both := errors.Join(context.Canceled, errors.New("disk full"))
	fmt.Println("cancelled and failed, every error must be clean:", cmdutil.IsCleanShutdown(both, false))
	fmt.Println("cancelled and failed, a cancellation is enough:", cmdutil.IsCleanShutdown(both, true))

	// A command names its own benign shutdown errors as clean.
	fmt.Println("server closed:", outcome(http.ErrServerClosed))
	fmt.Println("server closed, named clean:", cmdutil.IsCleanShutdown(http.ErrServerClosed, false, http.ErrServerClosed))

	// Output:
	// nil: clean
	// interrupt: clean
	// cancelled: clean
	// wrapped cancellation: clean
	// deadline exceeded: failed
	// failure: failed
	// cancelled and failed, every error must be clean: false
	// cancelled and failed, a cancellation is enough: true
	// server closed: failed
	// server closed, named clean: true
}

func TestIsCleanShutdownClean(t *testing.T) {
	failure := errors.New("something crashed")

	for _, tc := range []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"context.Canceled", context.Canceled},
		{"ErrInterrupt", cmdutil.ErrInterrupt},
		{"wrapped", fmt.Errorf("listening: %w", context.Canceled)},
		{"wrapped twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", cmdutil.ErrInterrupt))},
		{"joined cancellations", errors.Join(context.Canceled, cmdutil.ErrInterrupt)},
		{"errors.M", appendTo(context.Canceled)},
		{"errors.M with several", appendTo(context.Canceled, cmdutil.ErrInterrupt)},
		{"deadline is not a cancellation", context.DeadlineExceeded},
		{"failure", failure},
		{"wrapped failure", fmt.Errorf("listening: %w", failure)},
		{"failure alongside a cancellation", appendTo(failure, context.Canceled)},
		{"joined failure and cancellation", errors.Join(context.Canceled, failure)},
	} {
		// Both modes agree on everything except an aggregate that mixes a
		// failure with a cancellation, which is covered separately below.
		want := tc.err == nil ||
			errors.Is(tc.err, context.Canceled) || errors.Is(tc.err, cmdutil.ErrInterrupt)
		if errors.Is(tc.err, failure) {
			want = false
		}
		if got := cmdutil.IsCleanShutdown(tc.err, false); got != want {
			t.Errorf("%v: all-clean: got %v, want %v", tc.name, got, want)
		}
	}
}

// TestIsCleanShutdownCancelIsEnough covers the distinction the flag makes: an
// aggregate holding a real failure alongside a cancellation is clean only when
// a cancellation anywhere in the tree suffices.
func TestIsCleanShutdownCancelIsEnough(t *testing.T) {
	failure := errors.New("something crashed")
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"errors.M", appendTo(failure, context.Canceled)},
		{"errors.Join", errors.Join(failure, cmdutil.ErrInterrupt)},
		{"wrapped aggregate", fmt.Errorf("serving: %w", errors.Join(failure, context.Canceled))},
	} {
		if got := cmdutil.IsCleanShutdown(tc.err, true); !got {
			t.Errorf("%v: cancelIsEnough: got false, want true", tc.name)
		}
		if got := cmdutil.IsCleanShutdown(tc.err, false); got {
			t.Errorf("%v: all-clean: got true, want false", tc.name)
		}
	}
}

// TestIsCleanShutdownExtraSentinels covers the clean variadic, which lets a
// command treat its own shutdown errors as benign.
func TestIsCleanShutdownExtraSentinels(t *testing.T) {
	for _, cancelIsEnough := range []bool{false, true} {
		// Without being named, a command's own closed error is a failure.
		if cmdutil.IsCleanShutdown(errClosed, cancelIsEnough) {
			t.Errorf("cancelIsEnough=%v: got true, want false for an unnamed sentinel", cancelIsEnough)
		}
		if !cmdutil.IsCleanShutdown(errClosed, cancelIsEnough, errClosed) {
			t.Errorf("cancelIsEnough=%v: got false, want true for a named sentinel", cancelIsEnough)
		}
		wrapped := fmt.Errorf("reading: %w", errClosed)
		if !cmdutil.IsCleanShutdown(wrapped, cancelIsEnough, errClosed) {
			t.Errorf("cancelIsEnough=%v: got false, want true for a wrapped named sentinel", cancelIsEnough)
		}
	}

	// A named sentinel mixes with a cancellation like any other clean error.
	mixed := appendTo(errClosed, context.Canceled)
	if !cmdutil.IsCleanShutdown(mixed, false, errClosed) {
		t.Error("all-clean: got false, want true for a named sentinel beside a cancellation")
	}
	if cmdutil.IsCleanShutdown(mixed, false) {
		t.Error("all-clean: got true, want false when the sentinel is not named")
	}
}

// TestIsCleanShutdownWrapsNothing covers an error that reports itself as clean
// but unwraps to nil, and one that is a failure and does the same: the latter
// must not be reported clean on the strength of having nothing underneath it.
func TestIsCleanShutdownWrapsNothing(t *testing.T) {
	if !cmdutil.IsCleanShutdown(unwrapsToNil{context.Canceled}, false) {
		t.Error("got false, want true for a cancellation that unwraps to nil")
	}
	if cmdutil.IsCleanShutdown(unwrapsToNil{errors.New("crashed")}, false) {
		t.Error("got true, want false for a failure that unwraps to nil")
	}
}

// TestIsCleanShutdownNested covers aggregates nested inside aggregates and
// inside wrappers: the whole tree has to be walked, not just its top level.
func TestIsCleanShutdownNested(t *testing.T) {
	failure := errors.New("something crashed")
	for _, tc := range []struct {
		name                   string
		err                    error
		allClean, cancelEnough bool
	}{
		{
			"clean aggregate of aggregates",
			errors.Join(context.Canceled, appendTo(cmdutil.ErrInterrupt, context.Canceled)),
			true, true,
		},
		{
			"failure two levels down",
			errors.Join(context.Canceled, appendTo(context.Canceled, failure)),
			false, true,
		},
		{
			"failure under a wrapper inside an aggregate",
			appendTo(context.Canceled, fmt.Errorf("worker: %w", failure)),
			false, true,
		},
		{
			"aggregate of failures only",
			errors.Join(failure, errors.New("and another")),
			false, false,
		},
	} {
		if got := cmdutil.IsCleanShutdown(tc.err, false); got != tc.allClean {
			t.Errorf("%v: all-clean: got %v, want %v", tc.name, got, tc.allClean)
		}
		if got := cmdutil.IsCleanShutdown(tc.err, true); got != tc.cancelEnough {
			t.Errorf("%v: cancelIsEnough: got %v, want %v", tc.name, got, tc.cancelEnough)
		}
	}
}

// TestIsCleanShutdownSeveralSentinels covers supplying more than one sentinel,
// as a command with several benign shutdown errors of its own does.
func TestIsCleanShutdownSeveralSentinels(t *testing.T) {
	errDrained := errors.New("drained")
	clean := []error{errClosed, errDrained}

	for _, err := range []error{
		errClosed,
		errDrained,
		appendTo(errClosed, errDrained),
		errors.Join(errDrained, context.Canceled),
	} {
		if !cmdutil.IsCleanShutdown(err, false, clean...) {
			t.Errorf("%v: got false, want true", err)
		}
	}
	if cmdutil.IsCleanShutdown(appendTo(errClosed, errors.New("crashed")), false, clean...) {
		t.Error("got true, want false for a failure beside a named sentinel")
	}
}

// TestIsCleanShutdownCleanAggregate covers an error that reports itself as a
// cancellation while holding a failure: being clean at one node must not
// excuse what that node aggregates.
func TestIsCleanShutdownCleanAggregate(t *testing.T) {
	agg := cleanAggregate{errs: []error{context.Canceled, errors.New("crashed")}}
	if cmdutil.IsCleanShutdown(agg, false) {
		t.Error("all-clean: got true, want false for a clean aggregate holding a failure")
	}
	if !cmdutil.IsCleanShutdown(agg, true) {
		t.Error("cancelIsEnough: got false, want true")
	}
}

// cleanAggregate reports itself as a cancellation regardless of the errors it
// holds, as a custom aggregate type can.
type cleanAggregate struct{ errs []error }

func (c cleanAggregate) Error() string        { return "cancelled" }
func (c cleanAggregate) Is(target error) bool { return target == context.Canceled }
func (c cleanAggregate) Unwrap() []error      { return c.errs }

// unwrapsToNil reports err via Is but has nothing beneath it, as a custom
// error type that implements Unwrap without always wrapping something can.
type unwrapsToNil struct{ err error }

func (u unwrapsToNil) Error() string        { return u.err.Error() }
func (u unwrapsToNil) Is(target error) bool { return errors.Is(u.err, target) }
func (u unwrapsToNil) Unwrap() error        { return nil }

func appendTo(errs ...error) error {
	var m cerrors.M
	for _, err := range errs {
		m.Append(err)
	}
	return m.Err()
}

// TestIsCleanShutdownEdgeCases covers Go 1.20 multi-wrap %w, nil sentinels,
// and cyclic error references.
func TestIsCleanShutdownEdgeCases(t *testing.T) {
	failure := errors.New("crashed")

	// 1. Go 1.20 multiple %w verbs
	multiWrapFail := fmt.Errorf("%w: %w", context.Canceled, failure)
	if cmdutil.IsCleanShutdown(multiWrapFail, false) {
		t.Error("all-clean: got true, want false for multi-wrap with failure")
	}
	if !cmdutil.IsCleanShutdown(multiWrapFail, true) {
		t.Error("cancelIsEnough: got false, want true for multi-wrap with cancellation")
	}

	multiWrapClean := fmt.Errorf("%w: %w", context.Canceled, cmdutil.ErrInterrupt)
	if !cmdutil.IsCleanShutdown(multiWrapClean, false) {
		t.Error("all-clean: got false, want true for clean multi-wrap")
	}

	// 2. Nil sentinels in clean variadic
	if !cmdutil.IsCleanShutdown(context.Canceled, false, nil) {
		t.Error("all-clean: got false, want true with nil sentinel")
	}
	if cmdutil.IsCleanShutdown(failure, false, nil) {
		t.Error("all-clean: got true, want false for failure with nil sentinel")
	}

	// 3. Cyclic error references must not stack overflow or hang
	c1 := &cyclicErr{}
	c2 := &cyclicErr{next: c1}
	c1.next = c2
	if cmdutil.IsCleanShutdown(c1, false) {
		t.Error("all-clean: got true, want false for cyclic error")
	}
}

type cyclicErr struct {
	next error
}

func (c *cyclicErr) Error() string { return "cyclic" }
func (c *cyclicErr) Unwrap() error { return c.next }
