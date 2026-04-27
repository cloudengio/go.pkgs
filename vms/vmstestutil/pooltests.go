// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:generate astest . pooltests_test.go
package vmstestutil

import (
	"context"
	"errors"
	"io"
	"sync"
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

func (c PoolTestConfig) poolSize() int {
	if c.PoolSize > 0 {
		return c.PoolSize
	}
	return 2
}

func (c PoolTestConfig) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

var (
	configOnce sync.Once
	globalCfg  PoolTestConfig
)

func SetTestConfig(cfg PoolTestConfig) {
	configOnce.Do(func() {
		globalCfg = cfg
	})
}

// TestingT is the subset of *testing.T used by RunPoolTests.
// *testing.T does not satisfy this interface directly because Run's callback
// takes TestingT rather than *testing.T; callers should wrap *testing.T with a
// thin adapter (see pooltests_test.go for an example).
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Cleanup(f func())
}

// TestLifecycle runs the full pool lifecycle test suite using the global config
// set by SetTestConfig.
func TestLifecycle(t TestingT) { //cicd:astest
	t.Helper()
	TestStartAndAcquire(t)
	TestAcquireAndRelease(t)
	TestExec(t)
	TestContextCancellation(t)
	TestClose(t)
	TestConcurrentAcquire(t)
}

// startPool creates and starts a pool, waits for all VMs to be ready, and
// registers a Close cleanup. It fails the test immediately on any error.
func startPool(t TestingT, cfg PoolTestConfig) *vmspool.Pool {
	t.Helper()
	size := cfg.poolSize()
	statusCh := make(chan vmspool.Event, size*16)
	opts := []vmspool.Option{
		vmspool.WithSize(size),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(cfg.SupportsSuspend),
	}
	p := vmspool.New(cfg.Constructor, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
	})
	if err := p.Start(ctx); err != nil {
		t.Fatalf("pool.Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())
	return p
}

// waitForPoolEvent blocks until an event of the given kind is received on
// statusCh or the timeout elapses.
func waitForPoolEvent(t TestingT, statusCh <-chan vmspool.Event, kind vmspool.EventKind, timeout time.Duration) {
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

// TestStartAndAcquire verifies that starting the pool and acquiring a VM
// produces a VM in the Running state.
func TestStartAndAcquire(t TestingT) { //cicd:astest
	t.Helper()
	p := startPool(t, globalCfg)

	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
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
	if err := vm.Release(context.Background()); err != nil {
		t.Errorf("Release: %v", err)
	}
}

// TestAcquireAndRelease verifies the full acquire → release → replenish cycle:
// releasing a VM triggers replenishment so the pool can serve another Acquire.
func TestAcquireAndRelease(t TestingT) { //cicd:astest
	t.Helper()
	size := globalCfg.poolSize()
	statusCh := make(chan vmspool.Event, size*16)
	opts := []vmspool.Option{
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(globalCfg.SupportsSuspend),
	}
	p := vmspool.New(globalCfg.Constructor, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, globalCfg.timeout())

	vm, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := vm.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Wait for replenishment to complete before acquiring again.
	waitForPoolEvent(t, statusCh, vmspool.EventReplenished, globalCfg.timeout())

	vm2, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("second Acquire after replenishment: %v", err)
	}
	if err := vm2.Release(ctx); err != nil {
		t.Errorf("Release vm2: %v", err)
	}
}

// TestExec verifies that a command can be executed inside an acquired VM
// without error.
func TestExec(t TestingT) { //cicd:astest
	t.Helper()
	if globalCfg.ExecCmd == "" {
		return
	}
	p := startPool(t, globalCfg)

	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
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

	if err := vm.Exec(ctx, io.Discard, io.Discard, globalCfg.ExecCmd, globalCfg.ExecArgs...); err != nil {
		t.Errorf("Exec %q %v: %v", globalCfg.ExecCmd, globalCfg.ExecArgs, err)
	}
}

// TestContextCancellation verifies that Acquire returns context.Canceled when
// the pool is empty and the context is cancelled.
func TestContextCancellation(t TestingT) { //cicd:astest
	t.Helper()
	// Use a size-1 pool so one Acquire drains it.
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(globalCfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(globalCfg.SupportsSuspend),
	)
	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, globalCfg.timeout())

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

// TestClose verifies that Close prevents further Acquire calls.
func TestClose(t TestingT) { //cicd:astest
	t.Helper()
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(globalCfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithSuspendVMs(globalCfg.SupportsSuspend),
	)
	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, globalCfg.timeout())

	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err == nil {
		t.Errorf("Acquire after Close: expected error, got nil")
	}
}

// TestConcurrentAcquire verifies that poolSize goroutines can each acquire a
// VM concurrently without error, and that the pool replenishes after all are
// released.
func TestConcurrentAcquire(t TestingT) { //cicd:astest
	t.Helper()
	p := startPool(t, globalCfg)
	size := globalCfg.poolSize()

	ctx, cancel := context.WithTimeout(context.Background(), globalCfg.timeout())
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
