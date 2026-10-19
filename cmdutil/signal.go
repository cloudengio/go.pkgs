// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"cloudeng.io/sync/errgroup"
)

// ErrInterrupt is returned as the cause for HandleInterrupt cancellations.
var ErrInterrupt = errors.New("interrupted")

// HandleInterrupt returns a context that is cancelled when an interrupt
// signal is received. The returned CancelCauseFunc should be used to
// cancel the context and will return ErrInterrupt as the cause.
func HandleInterrupt(ctx context.Context) (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	HandleSignals(func() { cancel(ErrInterrupt) }, os.Interrupt)
	return ctx, cancel
}

// IsCleanShutdown reports whether err is the result of a clean,
// user-initiated shutdown, ie. one triggered by a signal handled via
// HandleInterrupt or HandleSignals, or by cancelling the command's context,
// rather than by a failure. It is intended for the top level of a CLI, to
// decide between exiting quietly and reporting an error:
//
//	if err := cli.Dispatch(ctx); err != nil {
//		if cmdutil.IsCleanShutdown(err, false) {
//			return
//		}
//		cmdutil.Exitf("error: %v", err)
//	}
//
// context.Canceled and ErrInterrupt always count as clean, as does any error
// supplied in clean, which lets a command add its own benign shutdown
// sentinels, eg. http.ErrServerClosed or a client's own 'closed' error.
//
// cancelIsEnough selects between the two readings of an error that aggregates
// several others. When true, err is clean if any error in its tree is clean,
// which suits a command whose shutdown path cannot report anything but a
// cancellation. When false, every error in the tree must be clean. The latter
// is what a command that collects errors across concurrent components needs:
// such a command typically aggregates ctx.Err() alongside whatever its
// components returned, so a run that was both cancelled and genuinely broken
// holds context.Canceled next to the real failure, and reporting it as clean
// would exit quietly on a real error.
func IsCleanShutdown(err error, cancelIsEnough bool, clean ...error) bool {
	if err == nil {
		return true
	}
	if cancelIsEnough {
		return anyClean(err, clean)
	}
	return allClean(err, clean)
}

// anyClean reports whether any error in err's tree is a clean shutdown error,
// ie. it answers the question that errors.Is is built for.
func anyClean(err error, clean []error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrInterrupt) {
		return true
	}
	for _, c := range clean {
		if c != nil && errors.Is(err, c) {
			return true
		}
	}
	return false
}

const maxErrorDepth = 100

// allClean reports whether every error in err's tree is a clean shutdown
// error. The standard library offers no way to enumerate an error tree --
// errors.Is and errors.As both answer "does any node match" -- so the two
// unwrap shapes that it recognises are handled here: Unwrap() []error, as
// implemented by errors.Join and similar aggregates, and Unwrap() error, as
// implemented by fmt.Errorf("...: %w", err).
func allClean(err error, clean []error) bool {
	return allCleanDepth(err, clean, 0)
}

func allCleanDepth(err error, clean []error, depth int) bool {
	if err == nil {
		return true
	}
	if depth > maxErrorDepth {
		// Guard against cyclic error chains causing stack overflow.
		return false
	}
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		errs := u.Unwrap()
		nonNil := 0
		for _, e := range errs {
			if e != nil {
				nonNil++
				if !allCleanDepth(e, clean, depth+1) {
					return false
				}
			}
		}
		if nonNil > 0 {
			return true
		}
	}
	if u, ok := err.(interface{ Unwrap() error }); ok {
		if child := u.Unwrap(); child != nil {
			return allCleanDepth(child, clean, depth+1)
		}
	}
	return isNodeClean(err, clean)
}

func isNodeClean(err error, clean []error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrInterrupt) {
		return true
	}
	for _, c := range clean {
		if c != nil && errors.Is(err, c) {
			return true
		}
	}
	return false
}

// HandleSignals will asynchronously invoke the supplied function when the
// specified signals are received.
func HandleSignals(fn func(), signals ...os.Signal) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	go func() {
		sig := <-sigCh
		fmt.Println("stopping on... ", sig)
		fn()
	}()
}

// Exitf formats and prints the supplied parameters to os.Stderr and then
// calls os.Exit(1).
func Exitf(format string, args ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// Exit formats and prints the supplied parameters to os.Stderr and then
// calls os.Exit(1).
//
// Deprecated: use Exitf instead.
func Exit(format string, args ...any) {
	Exitf(format, args...)
}

// WaitForExit waits for all provided functions to return
func WaitForExit(ctx context.Context, funcs ...func() error) error {
	g, _ := errgroup.WithContext(ctx)
	for _, fn := range funcs {
		g.Go(func() error {
			return fn()
		})
	}
	return g.Wait()
}

// WaitForExitCtx is like WaitForExit but the functions are passed the context
// that is cancelled when an error is returned by any of the functions.
func WaitForExitCtx(ctx context.Context, funcs ...func(context.Context) error) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, fn := range funcs {
		g.Go(func() error {
			return fn(ctx)
		})
	}
	return g.Wait()
}
