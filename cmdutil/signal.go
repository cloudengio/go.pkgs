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
