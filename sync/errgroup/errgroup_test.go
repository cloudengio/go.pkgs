// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errgroup_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/sync/errgroup"
)

func ExampleT() {
	// Wait for all goroutines to finish and catalogue all of their
	// errors.
	var g errgroup.T
	msg := []string{"a", "b", "c"}
	for _, m := range msg {
		m := m
		g.Go(func() error {
			return errors.New(m)
		})
	}
	err := g.Wait()
	if err == nil {
		fmt.Print("no errors - that's an error")
	}
	// Sort the error messages for stable output.
	out := strings.Split(err.Error(), "\n")
	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// --- 1 of 3 errors
	//   --- 2 of 3 errors
	//   --- 3 of 3 errors
	//   a
	//   b
	//   c
}

func ExampleT_parallel() {
	// Execute a set of gourtines in parallel.
	var g errgroup.T
	msg := []string{"a", "b", "c"}
	out := make([]string, len(msg))
	for i, m := range msg {
		i, m := i, m
		g.Go(func() error {
			out[i] = m
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		fmt.Printf("failed: %v", err)
	}
	// Sort the error messages for stable output.
	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// a
	// b
	// c
}

func ExampleWithContext() {
	// Terminate all remaining goroutines after a single error is encountered.
	g, ctx := errgroup.WithContext(context.Background())
	var msg = []string{"a", "b", "c"}
	for i, m := range msg {
		i, m := i, m
		g.Go(func() error {
			if i == 1 {
				return errors.New("first")
			}
			<-ctx.Done()
			return fmt.Errorf("%v: %w", m, ctx.Err())
		})
	}
	err := g.Wait()
	if err == nil {
		fmt.Print("no errors - that's an error")
	}
	// Sort the error messages for stable output.
	out := strings.Split(err.Error(), "\n")
	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// --- 1 of 3 errors
	//   --- 2 of 3 errors
	//   --- 3 of 3 errors
	//   a: context canceled
	//   c: context canceled
	//   first
}

func ExampleWithCancel() {
	// Exit all goroutines when a deadline has passed.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	g := errgroup.WithCancel(cancel)
	var msg = []string{"a", "b", "c"}
	for _, m := range msg {
		m := m
		g.Go(func() error {
			ctx.Done()
			// deadline is already past.
			return fmt.Errorf("%v: %w", m, ctx.Err())
		})
	}
	err := g.Wait()
	if err == nil {
		fmt.Print("no errors - that's an error")
	}
	// Sort the error messages for stable output.
	out := strings.Split(err.Error(), "\n")
	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// --- 1 of 3 errors
	//   --- 2 of 3 errors
	//   --- 3 of 3 errors
	//   a: context deadline exceeded
	//   b: context deadline exceeded
	//   c: context deadline exceeded
}

func testConcurrency(t *testing.T, concurrency int) {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	g = errgroup.WithConcurrency(g, concurrency)

	var started int64
	intCh := make(chan int64, 1)

	go func() {
		// This could be flaky, but in practice, 1 seconds should be massively
		// conservative for starting a small # of goroutines that immediately
		// call select.
		time.Sleep(time.Second)
		intCh <- atomic.LoadInt64(&started)
		cancel()
	}()

	invocations := 50
	for i := 0; i < invocations; i++ {
		g.Go(func() error {
			atomic.AddInt64(&started, 1)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Minute):
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		fmt.Printf("unexpected error: %v", err)
	}

	if concurrency == 0 {
		// No limit on concurrency, make sure we've started at least
		// half the requested invocations.
		if got, want := <-intCh, int64(invocations)/2; got < want {
			t.Errorf("got %v, want %v", got, want)
		}
	} else {
		if got, want := <-intCh, int64(concurrency); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	if got, want := atomic.LoadInt64(&started), int64(invocations); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLimit(t *testing.T) {
	testConcurrency(t, 2)
	// Test with no limit.
	testConcurrency(t, 0)
}

func ExampleT_pipeline() {
	// A pipeline to generate random numbers and measure the  uniformity of
	// their distribution. The pipeline runs for 1 second.
	// The use of errgroup.T ensures that on return all of the goroutines
	// have completed and the chanels used are closed.

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	g := errgroup.WithCancel(cancel)
	numGenerators, numCounters := 4, 8

	numCh := make(chan int64)
	src := rand.New(rand.NewSource(1234))
	var srcMu sync.Mutex

	// numGenerators goroutines produce random numbers in the range of 0..99.
	for i := 0; i < numGenerators; i++ {
		g.Go(func() error {
			for {
				srcMu.Lock()
				n := src.Int63n(100)
				srcMu.Unlock()
				select {
				case numCh <- n:
				case <-ctx.Done():
					err := ctx.Err()
					if errors.Is(err, context.DeadlineExceeded) {
						return nil
					}
					return err
				default:
					break
				}
			}
		})
	}

	counters := make([]int64, 10)
	var total int64

	// numCounters consume the random numbers and count which decile
	// each one falls into.
	for i := 0; i < numCounters; i++ {
		g.Go(func() error {
			for {
				select {
				case num := <-numCh:
					atomic.AddInt64(&counters[num%10], 1)
					atomic.AddInt64(&total, 1)
				case <-ctx.Done():
					err := ctx.Err()
					if errors.Is(err, context.DeadlineExceeded) {
						return nil
					}
					return err
				}
			}
		})
	}

	go func() {
		g.Wait()
		close(numCh)
	}()

	if err := g.Wait(); err != nil {
		fmt.Printf("failed: %v", err)
	}
	// After one second, measure the normalized number of random numbers
	// per decile with appropriate rounding. Print the distribution
	// to verify the expected values.
	for i, v := range counters {
		ratio := total / v
		if ratio == 9 {
			// 9 is close enough to an even distribution so round
			// it up to 10.
			ratio = 10
		}
		fmt.Printf("%v: %v\n", i, ratio)
	}
	// Output:
	// 0: 10
	// 1: 10
	// 2: 10
	// 3: 10
	// 4: 10
	// 5: 10
	// 6: 10
	// 7: 10
	// 8: 10
	// 9: 10
}
