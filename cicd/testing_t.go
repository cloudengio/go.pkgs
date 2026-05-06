// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// TestingT mirrors testing.T and is implemented by cicd.Testing.
type TestingT interface {
	Helper()
	Context() context.Context
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
	Name() string
	Failed() bool
	Skipped() bool
	Log(args ...any)
	Logf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Skip(args ...any)
	Cleanup(f func())
}

// Testing is a concrete implementation of TestingT for use outside the test
// harness (e.g. integration tests run as binaries). Fatal/Fatalf and
// Skip/Skipf terminate the current goroutine via runtime.Goexit, which runs
// deferred functions before exiting — matching the behaviour of *testing.T.
// Note that RunCleanups must be called to run registered cleanup functions after
// the test body completes, matching *testing.T semantics.
type Testing struct {
	ctx    context.Context
	cancel context.CancelFunc
	name   string

	mu       sync.Mutex
	failed   bool
	skipped  bool
	cleanups []func()

	outputMu sync.Mutex
	output   io.Writer
}

// NewTesting creates a Testing with the given name. Output goes to w;
// pass nil to use os.Stderr.
func NewTesting(ctx context.Context, name string, w io.Writer) *Testing {
	ctx, cancel := context.WithCancel(ctx)
	if w == nil {
		w = os.Stderr
	}
	return &Testing{ctx: ctx, cancel: cancel, name: name, output: w}
}

// Helper is a no-op; call-stack marking is not available outside the test harness.
func (t *Testing) Helper() {}

// Name returns the name set at construction.
func (t *Testing) Name() string { return t.name }

// Context returns the context for this test. The context is cancelled just
// before RunCleanups is called, matching testing.T.Context() semantics.
func (t *Testing) Context() context.Context { return t.ctx }

// Failed reports whether the test has been marked as failed.
func (t *Testing) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

// Skipped reports whether the test has been marked as skipped.
func (t *Testing) Skipped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.skipped
}

// Log writes a message to the output writer.
func (t *Testing) Log(args ...any) {
	t.log(fmt.Sprint(args...))
}

// Logf writes a formatted message to the output writer.
func (t *Testing) Logf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
}

// Error marks the test as failed and writes a message.
func (t *Testing) Error(args ...any) {
	t.log(fmt.Sprint(args...))
	t.markFailed()
}

// Errorf marks the test as failed and writes a formatted message.
func (t *Testing) Errorf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.markFailed()
}

// Fatal marks the test as failed, writes a message, then terminates the
// current goroutine via runtime.Goexit.
func (t *Testing) Fatal(args ...any) {
	t.log(fmt.Sprint(args...))
	t.markFailed()
	runtime.Goexit()
}

// Fatalf marks the test as failed, writes a formatted message, then terminates
// the current goroutine via runtime.Goexit.
func (t *Testing) Fatalf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.markFailed()
	runtime.Goexit()
}

// Skip marks the test as skipped, writes a message, then terminates the
// current goroutine via runtime.Goexit.
func (t *Testing) Skip(args ...any) {
	t.log(fmt.Sprint(args...))
	t.markSkipped()
	runtime.Goexit()
}

// Skipf marks the test as skipped, writes a formatted message, then terminates
// the current goroutine via runtime.Goexit.
func (t *Testing) Skipf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.markSkipped()
	runtime.Goexit()
}

// Cleanup registers a function to be called when RunCleanups is invoked.
// Functions are called in last-in-first-out order, matching *testing.T.
func (t *Testing) Cleanup(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanups = append(t.cleanups, f)
}

// Run mirrors testing.T.Run: it creates a child Testing named "parent/name",
// runs f in a new goroutine (so Fatal/Skip only exit the child), waits for
// completion. If the child fails, the parent is also marked as failed,
// matching testing.T.Run semantics. Returns true if the child did not fail.
func (t *Testing) Run(name string, f func(*Testing)) bool {
	child := NewTesting(t.ctx, strings.TrimPrefix(t.name+"/"+name, "/"), t.output)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer child.RunCleanups()
		f(child)
	}()
	<-done
	if child.Failed() {
		t.markFailed()
		return false
	}
	return true
}

// RunCleanups runs all registered cleanup functions in LIFO order and clears
// the cleanup list.
func (t *Testing) RunCleanups() {
	t.cancel()
	t.mu.Lock()
	fns := t.cleanups
	t.cleanups = nil
	t.mu.Unlock()
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

func (t *Testing) log(msg string) {
	t.outputMu.Lock()
	defer t.outputMu.Unlock()
	fmt.Fprintf(t.output, "%s: %s\n", t.name, msg)
}

func (t *Testing) markFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed = true
}

func (t *Testing) markSkipped() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.skipped = true
}

// TestMain runs each test in tests with its own fresh *Testing. T must be
// compatible with *Testing (i.e. *Testing or an interface it implements, such
// as TestingT). Each test's name is derived from its function name via
// reflection. Tests run in slice order. Output goes to w (nil → os.Stderr).
func TestMain[T TestingT](ctx context.Context, name string, w io.Writer, tests []func(T)) error {
	if w == nil {
		w = os.Stderr
	}
	failed := false
	for _, f := range tests {
		testName := funcBaseName(f)
		t := NewTesting(ctx, strings.TrimPrefix(name+"/"+testName, "/"), w)
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer t.RunCleanups()
			tt, ok := any(t).(T)
			if !ok {
				panic(fmt.Sprintf("cicd.TestMain: %T is not %T", t, tt))
			}
			f(tt)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
		if t.Failed() {
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("some tests failed")
	}
	return nil
}

// funcBaseName returns the unqualified function name of f, stripping the
// package path and any "-fm" method-value suffix added by the runtime.
func funcBaseName(f any) string {
	v := reflect.ValueOf(f)
	if !v.IsValid() || v.IsNil() {
		return "unknown"
	}
	full := runtime.FuncForPC(v.Pointer()).Name()
	if i := strings.LastIndexByte(full, '.'); i >= 0 {
		full = full[i+1:]
	}
	return strings.TrimSuffix(full, "-fm")
}
