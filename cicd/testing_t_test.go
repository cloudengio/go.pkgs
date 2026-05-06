// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
	"bytes"
	"strings"
	"testing"
)

// runInGoroutine runs f in a fresh goroutine and blocks until it completes,
// matching how *testing.T internally runs subtests so that runtime.Goexit
// only exits the goroutine running f, not the test goroutine.
func runInGoroutine(f func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	<-done
}

// TestTestingName checks that Name returns what was passed to NewTesting.
func TestTestingName(t *testing.T) {
	tt := NewTesting("mytest", nil)
	if got := tt.Name(); got != "mytest" {
		t.Errorf("Name: got %q, want %q", got, "mytest")
	}
}

// TestTestingInitialState matches *testing.T: a fresh T is neither failed nor skipped.
func TestTestingInitialState(t *testing.T) {
	tt := NewTesting("mytest", nil)
	if tt.Failed() {
		t.Error("Failed: want false for fresh Testing")
	}
	if tt.Skipped() {
		t.Error("Skipped: want false for fresh Testing")
	}
}

// TestTestingOutputFormat verifies the "name: message\n" line format.
func TestTestingOutputFormat(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	tt.Log("hello")
	if got, want := buf.String(), "mytest: hello\n"; got != want {
		t.Errorf("output: got %q, want %q", got, want)
	}
}

func TestTestingLog(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	tt.Log("hello", " world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Log output %q does not contain expected message", buf.String())
	}
	if tt.Failed() {
		t.Error("Log must not mark test as failed")
	}
	if tt.Skipped() {
		t.Error("Log must not mark test as skipped")
	}
}

func TestTestingLogf(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	tt.Logf("value=%d", 42)
	if !strings.Contains(buf.String(), "value=42") {
		t.Errorf("Logf output %q does not contain expected message", buf.String())
	}
	if tt.Failed() {
		t.Error("Logf must not mark test as failed")
	}
}

func TestTestingError(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	tt.Error("something went wrong")
	if !tt.Failed() {
		t.Error("Error must mark test as failed")
	}
	if tt.Skipped() {
		t.Error("Error must not mark test as skipped")
	}
	if !strings.Contains(buf.String(), "something went wrong") {
		t.Errorf("Error output %q does not contain expected message", buf.String())
	}
}

func TestTestingErrorf(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	tt.Errorf("bad value: %d", 99)
	if !tt.Failed() {
		t.Error("Errorf must mark test as failed")
	}
	if !strings.Contains(buf.String(), "bad value: 99") {
		t.Errorf("Errorf output %q does not contain expected message", buf.String())
	}
}

// TestTestingFatalExitsGoroutine mirrors testing.T: Fatal terminates the
// goroutine (via runtime.Goexit) so code after it never runs.
func TestTestingFatalExitsGoroutine(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	reached := false
	runInGoroutine(func() {
		tt.Fatal("stop here")
		reached = true //nolint:govet // intentionally unreachable
	})
	if reached {
		t.Error("code after Fatal must not execute")
	}
	if !tt.Failed() {
		t.Error("Fatal must mark test as failed")
	}
	if !strings.Contains(buf.String(), "stop here") {
		t.Errorf("Fatal output %q does not contain expected message", buf.String())
	}
}

func TestTestingFatalfExitsGoroutine(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	reached := false
	runInGoroutine(func() {
		tt.Fatalf("value=%d", 7)
		reached = true //nolint:govet // intentionally unreachable
	})
	if reached {
		t.Error("code after Fatalf must not execute")
	}
	if !tt.Failed() {
		t.Error("Fatalf must mark test as failed")
	}
	if !strings.Contains(buf.String(), "value=7") {
		t.Errorf("Fatalf output %q does not contain expected message", buf.String())
	}
}

// TestTestingFatalRunsDeferreds matches *testing.T: runtime.Goexit runs
// deferred functions, so Cleanup callbacks registered before Fatal still fire.
func TestTestingFatalRunsDeferreds(t *testing.T) {
	tt := NewTesting("mytest", nil)
	cleanupRan := false
	tt.Cleanup(func() { cleanupRan = true })
	runInGoroutine(func() {
		defer tt.RunCleanups()
		tt.Fatal("stop")
	})
	if !cleanupRan {
		t.Error("Cleanup registered before Fatal must still run (runtime.Goexit runs defers)")
	}
}

func TestTestingSkipExitsGoroutine(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	reached := false
	runInGoroutine(func() {
		tt.Skip("not applicable")
		reached = true //nolint:govet // intentionally unreachable
	})
	if reached {
		t.Error("code after Skip must not execute")
	}
	if !tt.Skipped() {
		t.Error("Skip must mark test as skipped")
	}
	if tt.Failed() {
		t.Error("Skip must not mark test as failed")
	}
	if !strings.Contains(buf.String(), "not applicable") {
		t.Errorf("Skip output %q does not contain expected message", buf.String())
	}
}

func TestTestingSkipfExitsGoroutine(t *testing.T) {
	var buf bytes.Buffer
	tt := NewTesting("mytest", &buf)
	reached := false
	runInGoroutine(func() {
		tt.Skipf("reason=%s", "ci")
		reached = true //nolint:govet // intentionally unreachable
	})
	if reached {
		t.Error("code after Skipf must not execute")
	}
	if !tt.Skipped() {
		t.Error("Skipf must mark test as skipped")
	}
	if !strings.Contains(buf.String(), "reason=ci") {
		t.Errorf("Skipf output %q does not contain expected message", buf.String())
	}
}

// TestTestingSkipRunsDeferreds matches Fatal: Skipf/Skip also use Goexit.
func TestTestingSkipRunsDeferreds(t *testing.T) {
	tt := NewTesting("mytest", nil)
	cleanupRan := false
	tt.Cleanup(func() { cleanupRan = true })
	runInGoroutine(func() {
		defer tt.RunCleanups()
		tt.Skip("skip")
	})
	if !cleanupRan {
		t.Error("Cleanup registered before Skip must still run")
	}
}

// TestTestingCleanupLIFO matches *testing.T: cleanups run in reverse registration order.
func TestTestingCleanupLIFO(t *testing.T) {
	tt := NewTesting("mytest", nil)
	var order []int
	tt.Cleanup(func() { order = append(order, 1) })
	tt.Cleanup(func() { order = append(order, 2) })
	tt.Cleanup(func() { order = append(order, 3) })
	tt.RunCleanups()
	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("Cleanup order: got %v, want [3 2 1]", order)
	}
}

// TestTestingRunCleanupsIdempotent matches *testing.T: a second RunCleanups
// call after the list is drained is a no-op.
func TestTestingRunCleanupsIdempotent(t *testing.T) {
	tt := NewTesting("mytest", nil)
	ran := 0
	tt.Cleanup(func() { ran++ })
	tt.RunCleanups()
	tt.RunCleanups()
	if ran != 1 {
		t.Errorf("RunCleanups ran %d times, want 1", ran)
	}
}

// TestTestingRunChildName matches testing.T.Run: subtest name is "parent/child".
func TestTestingRunChildName(t *testing.T) {
	tt := NewTesting("parent", nil)
	var childName string
	tt.Run("child", func(c *Testing) {
		childName = c.Name()
	})
	if childName != "parent/child" {
		t.Errorf("child Name: got %q, want %q", childName, "parent/child")
	}
}

// TestTestingRunReturnsFalseOnFailure matches testing.T.Run return value.
func TestTestingRunReturnsFalseOnFailure(t *testing.T) {
	tt := NewTesting("parent", nil)
	result := tt.Run("child", func(c *Testing) {
		c.Errorf("child failed")
	})
	if result {
		t.Error("Run must return false when child fails")
	}
}

func TestTestingRunReturnsTrueOnSuccess(t *testing.T) {
	tt := NewTesting("parent", nil)
	result := tt.Run("child", func(_ *Testing) {})
	if !result {
		t.Error("Run must return true when child passes")
	}
}

// TestTestingRunFatalIsolated matches testing.T.Run: Fatal in child only exits
// the child's goroutine, not the parent.
func TestTestingRunFatalIsolated(t *testing.T) {
	tt := NewTesting("parent", nil)
	reached := false
	result := tt.Run("child", func(c *Testing) {
		c.Fatal("child fatal")
		reached = true //nolint:govet // intentionally unreachable
	})
	if reached {
		t.Error("code after Fatal in child must not execute")
	}
	if result {
		t.Error("Run must return false when child calls Fatal")
	}
	if tt.Failed() {
		t.Error("parent must not be marked failed by child Fatal")
	}
}

// TestTestingRunCleanupRunsAfterF matches testing.T.Run: child cleanups run
// after f returns (or exits via Goexit).
func TestTestingRunCleanupRunsAfterF(t *testing.T) {
	tt := NewTesting("parent", nil)
	cleanupRan := false
	tt.Run("child", func(c *Testing) {
		c.Cleanup(func() { cleanupRan = true })
	})
	if !cleanupRan {
		t.Error("child Cleanup must run when Run returns")
	}
}

// TestTestingHelperNoOp verifies Helper does not panic (it's a no-op outside
// the test harness).
func TestTestingHelperNoOp(t *testing.T) {
	tt := NewTesting("mytest", nil)
	tt.Helper()
}

// TestTestingFuncBaseName checks reflection-based name extraction used by TestMain.
func TestTestingFuncBaseName(t *testing.T) {
	fn := func(_ *Testing) {}
	name := funcBaseName(fn)
	if name == "" {
		t.Error("funcBaseName returned empty string")
	}
	if strings.Contains(name, "/") || strings.Contains(name, ".") {
		t.Errorf("funcBaseName %q should be unqualified (no slashes or dots)", name)
	}
}
