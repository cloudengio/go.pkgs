// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	ExecCmd          string
	ExecArgs         []string
	ExecStdoutOutput string // Expected output from the exec.
	ExecStderrOutput string // Expected stderr output from the exec.

	StdoutRWC func(string) io.Writer // Optional factory
	StderrRWC func(string) io.Writer // Optional factory for stderr RWC used by Exec; defaults to bytes.Buffer-based implementation.

	// Timeout caps individual pool operations. Defaults to 30 s.
	Timeout time.Duration

	// StagingBehaviour determines the pool's staging behaviour. Defaults to
	// StagingBehaviourRunning.
	StagingBehaviour vmspool.StagingBehaviour
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

// TestingT is the subset of *testing.T used by RunPoolTests.
// *testing.T does not satisfy this interface directly because Run's callback
// takes TestingT rather than *testing.T; callers should wrap *testing.T with a
// thin adapter (see pooltests_test.go for an example).
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Cleanup(f func())
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
		vmspool.WithStagingBehaviour(cfg.StagingBehaviour),
	}
	p := vmspool.New(cfg.Constructor, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())

	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
		cancel()
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

// TestPoolAcquireExecRelease verifies the full acquire → exec → release → replenish cycle:
// releasing a VM triggers replenishment so the pool can serve another Acquire.
func TestPoolAcquireExecRelease(t TestingT, cfg PoolTestConfig) { //cicd:astest
	size := cfg.poolSize()
	statusCh := make(chan vmspool.Event, size*16)
	opts := []vmspool.Option{
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithStagingBehaviour(cfg.StagingBehaviour),
	}
	p := vmspool.New(cfg.Constructor, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	stdout := bytes.NewBuffer(make([]byte, 0, 1024))
	stderr := bytes.NewBuffer(make([]byte, 0, 1024))
	if err := vm.Exec(ctx, stdout, stderr, cfg.ExecCmd, cfg.ExecArgs...); err != nil {
		t.Errorf("Exec %q %v: %v", cfg.ExecCmd, cfg.ExecArgs, err)
	}
	if got, want := stdout.String(), cfg.ExecStdoutOutput; got != want {
		t.Errorf("Exec stdout: got %q, want %q", got, want)
	}
	if got, want := stderr.String(), cfg.ExecStderrOutput; got != want {
		t.Errorf("Exec stderr: got %q, want %q", got, want)
	}

	if err := vm.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if err := vm.Exec(ctx, stdout, stderr, cfg.ExecCmd, cfg.ExecArgs...); err == nil {
		t.Errorf("Exec %q %v: expected error, got nil", cfg.ExecCmd, cfg.ExecArgs)
	}

	// Wait for replenishment to complete before acquiring again.
	waitForPoolEvent(t, statusCh, vmspool.EventReplenished, cfg.timeout())

	vm2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("second Acquire after replenishment: %v", err)
	}

	stdout = bytes.NewBuffer(make([]byte, 0, 1024))
	stderr = bytes.NewBuffer(make([]byte, 0, 1024))
	if err := vm2.Exec(ctx, stdout, stderr, cfg.ExecCmd, cfg.ExecArgs...); err != nil {
		t.Errorf("Exec %q %v: %v", cfg.ExecCmd, cfg.ExecArgs, err)
	}
	if got, want := stdout.String(), cfg.ExecStdoutOutput; got != want {
		t.Errorf("Exec stdout: got %q, want %q", got, want)
	}
	if got, want := stderr.String(), cfg.ExecStderrOutput; got != want {
		t.Errorf("Exec stderr: got %q, want %q", got, want)
	}

	if err := vm2.Release(ctx); err != nil {
		t.Errorf("Release vm2: %v", err)
	}
}

// TestPoolContextCancellation verifies that Acquire returns context.Canceled when
// the pool is empty and the context is cancelled.
func TestPoolContextCancellation(t TestingT, cfg PoolTestConfig) { //cicd:astest
	// Use a size-1 pool so one Acquire drains it.
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(cfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithStagingBehaviour(cfg.StagingBehaviour),
	)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	// Drain the pool.
	vm, err := p.Acquire(ctx)
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
	_, err = p.Acquire(cancelCtx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Acquire with cancelled context: got %v, want context.Canceled", err)
	}
}

// TestPoolClose verifies that Close prevents further Acquire calls.
func TestPoolClose(t TestingT, cfg PoolTestConfig) { //cicd:astest
	statusCh := make(chan vmspool.Event, 16)
	p := vmspool.New(cfg.Constructor,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithStagingBehaviour(cfg.StagingBehaviour),
	)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForPoolEvent(t, statusCh, vmspool.EventStartPoolFull, cfg.timeout())

	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := p.Acquire(ctx)
	if err == nil {
		t.Errorf("Acquire after Close: expected error, got nil")
	}
}

// TestPoolConcurrentAcquire verifies that poolSize goroutines can each acquire a
// VM concurrently without error, and that the pool replenishes after all are
// released.
func TestPoolConcurrentAcquire(t TestingT, cfg PoolTestConfig) { //cicd:astest
	p := startPool(t, cfg)
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
			vm, err := p.Acquire(ctx)
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
