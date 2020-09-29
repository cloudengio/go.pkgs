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
	"sync"
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

// Handler represents a signal handler that can be used to wait for signal
// reception or context cancelation as per NotifyWithCancel. In addition
// it can be used to register additional cancel functions to be invoked
// on signal reception or context cancelation.
type Handler struct {
	retCh      chan os.Signal
	mu         sync.Mutex
	cancelList []func() // GUARDED_BY(mu)
}

// RegisterCancel registers one or more cancel functions to be invoked
// when a signal is received or the original context is canceled.
func (h *Handler) RegisterCancel(fns ...func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cancelList = append(h.cancelList, fns...)
}

func (h *Handler) cancel() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, cancel := range h.cancelList {
		cancel()
	}
}

// WaitForSignal will wait for a signal to be received. Context cancelation
// is translated into a ContextDoneSignal signal.
func (h *Handler) WaitForSignal() os.Signal {
	return <-h.retCh
}

// NotifyWithCancel is like signal.Notify except that it forks (and returns) the
// supplied context to obtain a cancel function that is called when a signal
// is received. It will also catch the cancelation of the supplied context
// and turn it into an instance of ContextDoneSignal. The returned handler can
// be used to wait for the signals to be received and to register additional
// cancelation functions to be invoked when a signal is received. Typical usage
// would be:
//
//   func main() {
//      ctx, handler := signals.NotifyWithCancel(context.Background(), signals.Defaults()...)
//      ....
//      handler.RegisterCancel(func() { ... })
//      ...
//      defer hanlder.WaitForSignal() // wait for a signal or context cancelation.
//    }
//
// If a second, different, signal is received then os.Exit(ExitCode) is called.
// Subsequent signals are the same as the first are ignored for one second
// but after that will similarly lead to os.Exit(ExitCode) being called.
func NotifyWithCancel(ctx context.Context, signals ...os.Signal) (context.Context, *Handler) {
	ctx, cancel := context.WithCancel(ctx)

	// Never drop the first two signals.
	notifyCh := make(chan os.Signal, 2)
	signal.Notify(notifyCh, signals...)
	isRunning := make(chan struct{})

	// Never block on forwarding the first signal.
	retCh := make(chan os.Signal, 1)
	handler := &Handler{retCh: retCh}
	go func() {
		close(isRunning)
		var sig os.Signal
		var sigTime time.Time
		select {
		case sig = <-notifyCh:
			sigTime = time.Now()
			retCh <- sig
			cancel()
			handler.cancel()
		case <-ctx.Done():
			retCh <- ContextDoneSignal(ctx.Err().Error())
			handler.cancel()
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
	return ctx, handler
}

// ContextDoneSignal implements os.Signal and is used to translate a
// canceled context into an os.Signal as forwarded by NotifyWithCancel.
type ContextDoneSignal string

// Signal implements os.Signal.
func (ContextDoneSignal) Signal() {}

// Stringimplements os.Signal.
func (s ContextDoneSignal) String() string { return string(s) }
