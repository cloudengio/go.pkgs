// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
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
	Fail()
	FailNow()
	SkipNow()
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
	helpers  map[string]struct{}

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

// Helper marks the calling function as a test helper function.
// When printing file and line information, that function will be skipped.
func (t *Testing) Helper() {
	var pc [1]uintptr
	if runtime.Callers(2, pc[:]) == 1 {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.helpers == nil {
			t.helpers = make(map[string]struct{})
		}
		frames := runtime.CallersFrames(pc[:])
		frame, _ := frames.Next()
		t.helpers[frame.Function] = struct{}{}
	}
}

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
	t.log(1, args...)
}

// Logf writes a formatted message to the output writer.
func (t *Testing) Logf(format string, args ...any) {
	t.logf(1, format, args...)
}

// Error marks the test as failed and writes a message.
func (t *Testing) Error(args ...any) {
	t.log(1, args...)
	t.markFailed()
}

// Errorf marks the test as failed and writes a formatted message.
func (t *Testing) Errorf(format string, args ...any) {
	t.logf(1, format, args...)
	t.markFailed()
}

// Fail marks the function as having failed but continues execution.
// It mirrors testing.T.Fail.
func (t *Testing) Fail() {
	t.markFailed()
}

// FailNow marks the function as having failed and stops its execution
// by calling runtime.Goexit (which runs all deferred calls in the current goroutine).
// It mirrors testing.T.FailNow.
func (t *Testing) FailNow() {
	t.markFailed()
	runtime.Goexit()
}

// Fatal marks the test as failed, writes a message, then terminates the
// current goroutine via runtime.Goexit.
func (t *Testing) Fatal(args ...any) {
	t.log(1, args...)
	t.markFailed()
	runtime.Goexit()
}

// Fatalf marks the test as failed, writes a formatted message, then terminates
// the current goroutine via runtime.Goexit.
func (t *Testing) Fatalf(format string, args ...any) {
	t.logf(1, format, args...)
	t.markFailed()
	runtime.Goexit()
}

// Skip marks the test as skipped, writes a message, then terminates the
// current goroutine via runtime.Goexit.
func (t *Testing) Skip(args ...any) {
	t.log(1, args...)
	t.markSkipped()
	runtime.Goexit()
}

// Skipf marks the test as skipped, writes a formatted message, then terminates
// the current goroutine via runtime.Goexit.
func (t *Testing) Skipf(format string, args ...any) {
	t.logf(1, format, args...)
	t.markSkipped()
	runtime.Goexit()
}

// SkipNow marks the test as skipped and stops its execution
// by calling runtime.Goexit.
// It mirrors testing.T.SkipNow.
func (t *Testing) SkipNow() {
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
		defer func() {
			if r := recover(); r != nil {
				child.Errorf("panic: %v", r)
			}
		}()
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
	for {
		t.mu.Lock()
		if len(t.cleanups) == 0 {
			t.mu.Unlock()
			break
		}
		fn := t.cleanups[len(t.cleanups)-1]
		t.cleanups = t.cleanups[:len(t.cleanups)-1]
		t.mu.Unlock()
		fn()
	}
}

func (t *Testing) caller(skip int) string {
	var pcs [16]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	t.mu.Lock()
	helpers := t.helpers
	t.mu.Unlock()
	for {
		frame, more := frames.Next()
		if helpers != nil {
			if _, ok := helpers[frame.Function]; ok {
				if !more {
					break
				}
				continue
			}
		}
		file := frame.File
		if idx := strings.LastIndexByte(file, '/'); idx >= 0 {
			file = file[idx+1:]
		}
		return fmt.Sprintf("%s:%d", file, frame.Line)
	}
	return "unknown:0"
}

func (t *Testing) logf(skip int, format string, args ...any) {
	t.outputMu.Lock()
	defer t.outputMu.Unlock()
	out := strings.Builder{}
	fmt.Fprintf(&out, "%s: %s: %s\n", t.caller(skip+1), t.name, fmt.Sprintf(format, args...))
	_, _ = t.output.Write([]byte(out.String()))
}

func (t *Testing) log(skip int, args ...any) {
	t.outputMu.Lock()
	defer t.outputMu.Unlock()
	out := strings.Builder{}
	fmt.Fprintf(&out, "%s: %s: %s\n", t.caller(skip+1), t.name, fmt.Sprint(args...))
	_, _ = t.output.Write([]byte(out.String()))
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
func TestMain[T TestingT](ctx context.Context, verbose bool, regex *regexp.Regexp, wr io.Writer, tests []func(T)) error {
	if wr == nil {
		wr = os.Stdout
	}
	failed := false
	failures := []string{}
	for _, f := range tests {
		testName := funcBaseName(f)
		if regex != nil && !regex.MatchString(testName) {
			continue
		}
		out := bytes.NewBuffer(make([]byte, 0, 16*1024))
		t := NewTesting(ctx, testName, out)
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer t.RunCleanups()
			start := time.Now()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic: %v", r)
				}
				took := time.Since(start)
				if t.Failed() {
					fmt.Fprintf(wr, "--- FAIL: %s (%v)\n", t.Name(), took)
					_, _ = out.WriteTo(wr)
				} else if verbose {
					fmt.Fprintf(wr, "--- PASS: %s (%v)\n", t.Name(), took)
					_, _ = out.WriteTo(wr)
				}
			}()
			tt, ok := any(t).(T)
			if !ok {
				panic(fmt.Sprintf("cicd.TestMain: %T is not %T", t, tt))
			}
			if verbose {
				fmt.Fprintf(wr, "=== RUN   %s\n", t.Name())
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
			failures = append(failures, t.Name())
		}
	}
	if failed {
		fmt.Fprintf(wr, "FAIL\n")
		fmt.Fprintf(wr, "Failed tests: %s\n", strings.Join(failures, ", "))
		return fmt.Errorf("failed tests: %s", strings.Join(failures, ", "))
	}
	fmt.Fprintf(wr, "PASS\n")
	return nil
}

var anonymousFuncRE = regexp.MustCompile(`\.(func\d+(?:\.\d+)*)$`)

// funcBaseName returns the unqualified function name of f, stripping the
// package path and any "-fm" method-value suffix added by the runtime.
func funcBaseName(f any) string {
	v := reflect.ValueOf(f)
	if !v.IsValid() || v.IsNil() {
		return "unknown"
	}
	full := runtime.FuncForPC(v.Pointer()).Name()
	full = strings.TrimSuffix(full, "-fm")

	if match := anonymousFuncRE.FindStringSubmatch(full); match != nil {
		return match[1]
	}

	if i := strings.LastIndexByte(full, '.'); i >= 0 {
		full = full[i+1:]
	}
	return full
}
