// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package signals provides support for working with operating system
// signals and contexts.
package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Defaults returns a set of platform specific signals that are commonly used.
func Defaults() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT}
}

const (
	// ExitCode is the exit code passed to os.Exit when a subsequent signal is
	// received.
	ExitCode = 1
)

// DebounceDuration is the time period during which subsequent identical
// signals are ignored.
var DebounceDuration time.Duration = time.Second

// NotifyWithCancel is like signal.Notify except that it forks (and returns) the
// supplied context to obtain a cancel function that is called when a signal
// is received. It will also catch the cancelation of the supplied context
// and turn into an instance of ContextDoneSignal. The returned function can
// be used to wait for the signals to be received, a function is returned to
// allow for the convenient use of defer. Typical usage would be:
//
// func main() {
//    ctx, wait := signals.NotifyWithCancel(context.Background(), signals.Defaults()...)
//    ....
//    defer wait() // wait for a signal or context cancelation.
//  }
//
// If a second, different, signal is received then os.Exit(ExitCode) is called.
// Subsequent signals are the same as the first are ignored for one second
// but after that will similarly lead to os.Exit(ExitCode) being called.
func NotifyWithCancel(ctx context.Context, signals ...os.Signal) (context.Context, func() os.Signal) {
	ctx, cancel := context.WithCancel(ctx)

	// Never drop the first two signals.
	notifyCh := make(chan os.Signal, 2)
	signal.Notify(notifyCh, signals...)
	isRunning := make(chan struct{})

	// Never block on forwarding the first signal.
	retCh := make(chan os.Signal, 1)
	go func() {
		close(isRunning)
		var sig os.Signal
		var sigTime time.Time
		select {
		case sig = <-notifyCh:
			sigTime = time.Now()
			retCh <- sig
			cancel()
		case <-ctx.Done():
			retCh <- ContextDoneSignal(ctx.Err().Error())
			return
		}
		for {
			subsequentSig := <-notifyCh
			if subsequentSig.String() != sig.String() || time.Since(sigTime) > DebounceDuration {
				os.Exit(ExitCode)
			}
		}
	}()
	<-isRunning
	return ctx, func() os.Signal {
		return <-retCh
	}
}

// ContextDoneSignal implements os.Signal and is used to translate a
// canceled context into an os.Signal as forwarded by NotifyWithCancel.
type ContextDoneSignal string

// Signal implements os.Signal.
func (ContextDoneSignal) Signal() {}

// Stringimplements os.Signal.
func (s ContextDoneSignal) String() string { return string(s) }
