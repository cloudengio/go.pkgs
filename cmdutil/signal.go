// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
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

// Exit formats and prints the supplied parameters to os.Stderr and then
// calls os.Exit(1).
func Exit(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// BuildInfoJSON returns the build information as a JSON raw message
// or nil if the build information is not available.
func BuildInfoJSON() json.RawMessage {
	if bi, ok := debug.ReadBuildInfo(); ok {
		d, _ := json.Marshal(bi)
		return d
	}
	return nil
}
