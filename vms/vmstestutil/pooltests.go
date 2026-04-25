// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"cloudeng.io/vms/vmspool"
)

// PoolTestConfig configures the pool integration test suite run by RunPoolTests.
type PoolTestConfig struct {
	// Constructor creates new VM instances. Required.
	Constructor vmspool.Constructor

	// PoolSize is the default pool size used across all tests. Defaults to 2.
	// Some subtests intentionally use a size-1 pool for deterministic behavior.
	PoolSize int

	// ExecCmd is a command that should succeed inside an acquired VM. If empty
	// the Exec subtest is skipped.
	ExecCmd  string
	ExecArgs []string

	// Timeout caps individual pool operations. Defaults to 30 s.
	Timeout time.Duration

	// SupportsSuspend enables the suspend-mode subtests (WithSuspendVMs(true)).
	// Set this when the constructor produces instances that support Suspend.
	SupportsSuspend bool
}

func (c *PoolTestConfig) poolSize() int {
	if c.PoolSize > 0 {
		return c.PoolSize
	}
	return 2
}

func (c *PoolTestConfig) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

// TestingT is the subset of *testing.T used by RunPoolTests.
// *testing.T satisfies this interface directly.
type TestingT interface {
	Helper()
	Run(name string, f func(*testing.T)) bool
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Cleanup(f func())
}

// RunPoolTests runs a suite of integration tests that verify a
// vmspool.Constructor creates instances that work correctly with vmspool.Pool.
// Each scenario runs as a t.Run subtest so results appear individually and
// failures are isolated.
//
// Callers must provide a Constructor whose instances are real (or sufficiently
// realistic) implementations of vms.Instance. The tests exercise the full
// acquire → exec → release → replenish lifecycle.
func RunPoolTests(t TestingT, cfg PoolTestConfig) {
	t.Helper()
	if cfg.SupportsSuspend {
		t.Run("SuspendMode", func(t *testing.T) {
			runSuspendModeTests(t, cfg)
		})
	}
	t.Run("RunningMode", func(t *testing.T) {
		runRunningModeTests(t, cfg)
	})
}

// runSuspendModeTests exercises pool behaviour with WithSuspendVMs(true):
// VMs are cloned, started, then suspended before being placed in the ready
// queue. Acquire re-starts each VM before handing it to the caller.
func runSuspendModeTests(t *testing.T, cfg PoolTestConfig) {
	t.Helper()
	t.Run("StartAndAcquire", func(t *testing.T) {
		testStartAndAcquire(t, cfg, true)
	})
	t.Run("AcquireAndRelease", func(t *testing.T) {
		testAcquireAndRelease(t, cfg, true)
	})
	if cfg.ExecCmd != "" {
		t.Run("Exec", func(t *testing.T) {
			testExec(t, cfg, true)
		})
	}
	t.Run("ContextCancellation", func(t *testing.T) {
		testContextCancellation(t, cfg, true)
	})
	t.Run("Close", func(t *testing.T) {
		testClose(t, cfg, true)
	})
	if cfg.poolSize() > 1 {
		t.Run("ConcurrentAcquire", func(t *testing.T) {
			testConcurrentAcquire(t, cfg, true)
		})
	}
}

// runRunningModeTests exercises pool behaviour with WithSuspendVMs(false):
// VMs are cloned and started (but not suspended) before being placed in the
// ready queue. Acquire hands the running VM directly to the caller.
func runRunningModeTests(t *testing.T, cfg PoolTestConfig) {
	t.Helper()
	t.Run("StartAndAcquire", func(t *testing.T) {
		testStartAndAcquire(t, cfg, false)
	})
	t.Run("AcquireAndRelease", func(t *testing.T) {
		testAcquireAndRelease(t, cfg, false)
	})
	if cfg.ExecCmd != "" {
		t.Run("Exec", func(t *testing.T) {
			testExec(t, cfg, false)
		})
	}
	t.Run("ContextCancellation", func(t *testing.T) {
		testContextCancellation(t, cfg, false)
	})
	t.Run("Close", func(t *testing.T) {
		testClose(t, cfg, false)
	})
	if cfg.poolSize() > 1 {
		t.Run("ConcurrentAcquire", func(t *testing.T) {
			testConcurrentAcquire(t, cfg, false)
		})
	}
}

// startPool creates and starts a pool, waits for all VMs to be ready, and
// registers a Close cleanup. It fails the test immediately on any error.
func startPool(t *testing.T, cfg PoolTestConfig, suspend bool) *vmspool.Pool {
	t.Helper()
	size := cfg.poolSize()
	statusCh := make(chan vmspool.Event, size*16)
	opts := []vmspool.Option{
		vmspool.WithSize(size),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(suspend),
	}
	p := vmspool.New(cfg.Constructor, opts...)
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("pool.Start: %v", err)
	}
	return p
}

// waitForPoolEvent blocks until an event of the given kind is received on
// statusCh or the timeout elapses.
func waitForPoolEvent(t *testing.T, statusCh <-chan vmspool.Event, kind vmspool.EventKind, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case e := <-statusCh:
			if e.Kind == kind {
				return
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for pool event %s", kind)
		}
	}
}

// testStartAndAcquire verifies that starting the pool and acquiring a VM
// produces a VM in the Running state.
func testStartAndAcquire(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	p := startPool(t, cfg, suspend)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	vm, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() {
		if err := vm.Release(context.Background()); err != nil {
			t.Errorf("Release: %v", err)
		}
	}()

}

// testAcquireAndRelease verifies the full acquire → release → replenish cycle:
// releasing a VM triggers replenishment so the pool can serve another Acquire.
func testAcquireAndRelease(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	size := cfg.poolSize()
	statusCh := make(chan vmspool.Event, size*16)
	opts := []vmspool.Option{
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(suspend),
	}
	p := vmspool.New(cfg.Constructor, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	vm, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := vm.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Wait for replenishment to complete before acquiring again.
	waitForPoolEvent(t, statusCh, vmspool.EventReplenished, cfg.timeout())

	vm2, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("second Acquire after replenishment: %v", err)
	}
	if err := vm2.Release(ctx); err != nil {
		t.Errorf("Release vm2: %v", err)
	}
}

// testExec verifies that a command can be executed inside an acquired VM
// without error.
func testExec(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	p := startPool(t, cfg, suspend)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	vm, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() {
		if err := vm.Release(context.Background()); err != nil {
			t.Errorf("Release: %v", err)
		}
	}()

	if err := vm.Exec(ctx, io.Discard, io.Discard, cfg.ExecCmd, cfg.ExecArgs...); err != nil {
		t.Errorf("Exec %q %v: %v", cfg.ExecCmd, cfg.ExecArgs, err)
	}
}

// testContextCancellation verifies that Acquire returns context.Canceled when
// the pool is empty and the context is cancelled.
func testContextCancellation(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	// Use a size-1 pool so one Acquire drains it.
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(cfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(suspend),
	)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	// Drain the pool.
	vm, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire (drain): %v", err)
	}
	defer func() {
		if err := vm.Release(context.Background()); err != nil {
			t.Errorf("Release: %v", err)
		}
	}()

	// Acquire on empty pool with pre-cancelled context must return immediately.
	cancelCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()
	_, err = p.Acquire(cancelCtx, io.Discard, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Acquire with cancelled context: got %v, want context.Canceled", err)
	}
}

// testClose verifies that Close prevents further Acquire calls.
func testClose(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(cfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(suspend),
	)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
	})
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err == nil {
		t.Error("Acquire after Close: expected error, got nil")
	}
}

// testConcurrentAcquire verifies that poolSize goroutines can each acquire a
// VM concurrently without error, and that the pool replenishes after all are
// released.
func testConcurrentAcquire(t *testing.T, cfg PoolTestConfig, suspend bool) {
	t.Helper()
	p := startPool(t, cfg, suspend)
	size := cfg.poolSize()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	type result struct {
		vm  *vmspool.VM
		err error
	}
	results := make([]result, size)
	done := make(chan int, size)
	for i := range size {
		go func(i int) {
			vm, err := p.Acquire(ctx, io.Discard, io.Discard)
			results[i] = result{vm, err}
			done <- i
		}(i)
	}
	for range size {
		<-done
	}

	for i, r := range results {
		if r.err != nil {
			t.Errorf("concurrent Acquire[%d]: %v", i, r.err)
		}
	}
	for i, r := range results {
		if r.vm == nil {
			continue
		}
		if err := r.vm.Release(context.Background()); err != nil {
			t.Errorf("Release[%d]: %v", i, err)
		}
	}
}
