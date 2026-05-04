// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxsync_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/sync/ctxsync"
)

func TestSingleFlight_Basic(t *testing.T) {
	sf := ctxsync.New()

	var calls int32
	v, err, shared := sf.Do(context.Background(), "key", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		return "success", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "success" {
		t.Errorf("got %v, want 'success'", v)
	}
	if shared {
		t.Errorf("expected shared=false")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestSingleFlight_Do_SharedCancel(t *testing.T) {
	sf := ctxsync.New()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	var calls int32
	fn1Started := make(chan struct{})
	fn2Done := make(chan struct{})

	go func() {
		<-fn1Started // wait for the first request to enter the func

		ctx2Ready := make(chan struct{})
		go func() {
			close(ctx2Ready)
			v, err, _ := sf.Do(ctx2, "key-do-cancel", func() (any, error) {
				atomic.AddInt32(&calls, 1)
				return "success2", nil
			})
			if err != nil {
				t.Errorf("ctx2 expected success, got error: %v", err)
			}
			if v != "success2" {
				t.Errorf("ctx2 expected 'success2', got %v", v)
			}
			close(fn2Done)
		}()

		<-ctx2Ready
		// Give ctx2 a moment to enter sf.Do and join the shared flight
		time.Sleep(20 * time.Millisecond)
		cancel1() // Cancel the first caller to trigger shared failure
	}()

	v, err, shared := sf.Do(ctx1, "key-do-cancel", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		close(fn1Started)
		<-ctx1.Done()
		return nil, ctx1.Err()
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("ctx1 expected context.Canceled, got %v", err)
	}
	if !shared {
		t.Errorf("ctx1 expected shared=true")
	}
	if v != nil {
		t.Errorf("ctx1 expected nil value, got %v", v)
	}

	<-fn2Done
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls due to retry, got %d", got)
	}
}

func TestSingleFlight_Do_BothCanceled(t *testing.T) {
	sf := ctxsync.New()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	var calls int32
	fn1Started := make(chan struct{})
	fn2Done := make(chan struct{})

	go func() {
		<-fn1Started

		ctx2Ready := make(chan struct{})
		go func() {
			close(ctx2Ready)
			_, err, _ := sf.Do(ctx2, "key-both-cancel", func() (any, error) {
				t.Error("should not be retried")
				return nil, nil
			})
			if !errors.Is(err, context.Canceled) {
				t.Errorf("ctx2 expected context.Canceled, got: %v", err)
			}
			close(fn2Done)
		}()

		<-ctx2Ready
		time.Sleep(20 * time.Millisecond)
		cancel2() // Cancel the second caller as well
		cancel1() // Cause the original caller to fail
	}()

	_, err, _ := sf.Do(ctx1, "key-both-cancel", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		close(fn1Started)
		<-ctx1.Done()
		return nil, ctx1.Err()
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("ctx1 expected context.Canceled, got %v", err)
	}

	<-fn2Done
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 call without retry, got %d", got)
	}
}

func TestSingleFlight_DoChan_Basic(t *testing.T) {
	sf := ctxsync.New()

	var calls int32
	ch := sf.DoChan(context.Background(), "key-chan-basic", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		return "success", nil
	})

	res := <-ch
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Val != "success" {
		t.Errorf("got %v, want 'success'", res.Val)
	}
	if res.Shared {
		t.Errorf("expected shared=false")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestSingleFlight_DoChan_SharedCancel(t *testing.T) {
	sf := ctxsync.New()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	var calls int32
	fn1Started := make(chan struct{})
	fn2Done := make(chan struct{})

	go func() {
		<-fn1Started

		ctx2Ready := make(chan struct{})
		go func() {
			ch := sf.DoChan(ctx2, "key-chan", func() (any, error) {
				atomic.AddInt32(&calls, 1)
				return "success2", nil
			})
			close(ctx2Ready)
			res := <-ch
			if res.Err != nil {
				t.Errorf("ctx2 expected success, got error: %v", res.Err)
			}
			if res.Val != "success2" {
				t.Errorf("ctx2 expected 'success2', got %v", res.Val)
			}
			close(fn2Done)
		}()

		<-ctx2Ready
		time.Sleep(20 * time.Millisecond)
		cancel1()
	}()

	ch := sf.DoChan(ctx1, "key-chan", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		close(fn1Started)
		<-ctx1.Done()
		return nil, ctx1.Err()
	})

	res := <-ch
	if !errors.Is(res.Err, context.Canceled) {
		t.Errorf("ctx1 expected context.Canceled, got %v", res.Err)
	}

	<-fn2Done
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls due to retry, got %d", got)
	}
}

func TestSingleFlight_DoChan_ContextCanceled(t *testing.T) {
	sf := ctxsync.New()

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	finished := make(chan struct{})

	ch := sf.DoChan(ctx, "key-chan-cancel", func() (any, error) {
		close(started)
		<-finished
		return "success", nil
	})

	<-started
	cancel()

	res := <-ch
	if !errors.Is(res.Err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", res.Err)
	}

	close(finished) // cleanup underlying goroutine
}

func TestSingleFlight_DoChan_BothCanceled(t *testing.T) {
	sf := ctxsync.New()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	var calls int32
	fn1Started := make(chan struct{})
	fn2Done := make(chan struct{})

	go func() {
		<-fn1Started

		ctx2Ready := make(chan struct{})
		go func() {
			ch := sf.DoChan(ctx2, "key-both-cancel-chan", func() (any, error) {
				t.Error("should not be retried")
				return nil, nil
			})
			close(ctx2Ready)
			res := <-ch
			if !errors.Is(res.Err, context.Canceled) {
				t.Errorf("ctx2 expected context.Canceled, got: %v", res.Err)
			}
			close(fn2Done)
		}()

		<-ctx2Ready
		time.Sleep(20 * time.Millisecond)
		cancel2() // Cancel the second caller as well
		cancel1() // Cause the original caller to fail
	}()

	ch := sf.DoChan(ctx1, "key-both-cancel-chan", func() (any, error) {
		atomic.AddInt32(&calls, 1)
		close(fn1Started)
		<-ctx1.Done()
		return nil, ctx1.Err()
	})

	res := <-ch
	if !errors.Is(res.Err, context.Canceled) {
		t.Errorf("ctx1 expected context.Canceled, got %v", res.Err)
	}

	<-fn2Done
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 call without retry, got %d", got)
	}
}

func TestSingleFlight_Forget(t *testing.T) {
	sf := ctxsync.New()

	var calls int32
	ch := make(chan struct{})
	fn1Started := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sf.Do(context.Background(), "key-forget", func() (any, error) {
			atomic.AddInt32(&calls, 1)
			close(fn1Started)
			<-ch
			return nil, nil
		})
	}()

	<-fn1Started
	sf.Forget("key-forget")

	fn2Started := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		sf.Do(context.Background(), "key-forget", func() (any, error) {
			atomic.AddInt32(&calls, 1)
			close(fn2Started)
			return nil, nil
		})
	}()

	<-fn2Started
	close(ch)
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls because of Forget, got %d", got)
	}
}
