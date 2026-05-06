// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
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
	TestingTSkip
	Failed() bool
	Skipped() bool
	Log(args ...any)
	Logf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Skip(args ...any)
	Cleanup(f func())
	RunCleanups()
	Run(name string, f func(*Testing)) bool
}

// Testing is a concrete implementation of TestingT for use outside the test
// harness (e.g. integration tests run as binaries). Fatal/Fatalf and
// Skip/Skipf terminate the current goroutine via runtime.Goexit, which runs
// deferred functions before exiting — matching the behaviour of *testing.T.
type Testing struct {
	mu       sync.Mutex
	name     string
	failed   bool
	skipped  bool
	output   io.Writer
	cleanups []func()
}

// NewTesting creates a Testing with the given name. Output goes to w;
// pass nil to use os.Stderr.
func NewTesting(name string, w io.Writer) *Testing {
	if w == nil {
		w = os.Stderr
	}
	return &Testing{name: name, output: w}
}

// Helper is a no-op; call-stack marking is not available outside the test harness.
func (t *Testing) Helper() {}

// Name returns the name set at construction.
func (t *Testing) Name() string { return t.name }

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
// completion, and returns true if the child did not fail.
func (t *Testing) Run(name string, f func(*Testing)) bool {
	child := NewTesting(t.name+"/"+name, t.output)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer child.RunCleanups()
		f(child)
	}()
	<-done
	return !child.Failed()
}

// RunCleanups runs all registered cleanup functions in LIFO order and clears
// the cleanup list.
func (t *Testing) RunCleanups() {
	t.mu.Lock()
	fns := t.cleanups
	t.cleanups = nil
	t.mu.Unlock()
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

func (t *Testing) log(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
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
// TestMain exits the process: 0 if all tests pass, 1 if any fail.
func TestMain[T TestingT](name string, w io.Writer, tests []func(T)) {
	if w == nil {
		w = os.Stderr
	}
	failed := false
	for _, f := range tests {
		testName := funcBaseName(f)
		t := NewTesting(name+"/"+testName, w)
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer t.RunCleanups()
			f(any(t).(T))
		}()
		<-done
		if t.Failed() {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// funcBaseName returns the unqualified function name of f, stripping the
// package path and any "-fm" method-value suffix added by the runtime.
func funcBaseName(f any) string {
	full := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	if i := strings.LastIndexByte(full, '.'); i >= 0 {
		full = full[i+1:]
	}
	return strings.TrimSuffix(full, "-fm")
}
