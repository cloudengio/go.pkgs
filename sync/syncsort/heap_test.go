// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package syncsort_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand" // #nosec: G404
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"cloudeng.io/sync/syncsort"
	"cloudeng.io/sync/synctestutil"
)

func generateRandomBytes(size int) []byte {
	data := make([]byte, size)
	n, err := rand.Reader.Read(data)
	if err != nil {
		panic(err)
	}
	if n != size {
		panic(fmt.Sprintf("got %v, want %v", n, size))
	}
	return data
}

func partitionSlices[T any](ctx context.Context, t *testing.T, seq *syncsort.Sequencer[[]T], data []T, blockSize int) []syncsort.Item[[]T] {
	t.Helper()
	dataSize := len(data)
	numBlocks := (dataSize / blockSize) + 1
	sblocks := make([]syncsort.Item[[]T], numBlocks)
	for i := range numBlocks {
		size := blockSize
		offset := i * size
		if offset+size > dataSize {
			size = dataSize - offset
		}
		item, err := seq.NextItem(ctx, data[offset:offset+size])
		if err != nil {
			t.Fatalf("NextItem failed: %v", err)
		}
		sblocks[i] = item
	}
	return sblocks
}

func shuffle[T any](items []syncsort.Item[T]) []syncsort.Item[T] {
	numItems := len(items)
	indices := map[int]bool{}
	rnd := mrand.New(mrand.NewSource(time.Now().UnixNano())) // #nosec: G404
	for {
		guess := rnd.Intn(numItems)
		if _, ok := indices[guess]; !ok {
			indices[guess] = true
		}
		if len(indices) == numItems {
			break
		}
	}
	shuffled := make([]syncsort.Item[T], 0, numItems)
	for idx := range indices {
		shuffled = append(shuffled, items[idx])
	}
	return shuffled
}

func flatten[T any](shuffled []syncsort.Item[[]T]) []T {
	flattened := make([]T, 0, len(shuffled))
	for _, s := range shuffled {
		flattened = append(flattened, s.V...)
	}
	return flattened
}

func send[T any](shuffled []syncsort.Item[T], ch chan<- syncsort.Item[T]) {
	for _, s := range shuffled {
		ch <- s
	}
	close(ch)
}

func TestStreamingSortByteSlice(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ich := make(chan syncsort.Item[[]byte], 10)
	testdata := generateRandomBytes(4096)

	seq := syncsort.NewSequencer(ctx, ich)
	blocks := partitionSlices(ctx, t, seq, testdata, 33)

	shuffled := shuffle(blocks)

	flattened := flatten(shuffled)
	if bytes.Equal(testdata, flattened) {
		t.Fatalf("shuffle failed")
	}

	go send(shuffled, ich)

	reordered := []byte{}
	for seq.Scan() {
		item := seq.Item()
		reordered = append(reordered, item...)
	}
	if err := seq.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := reordered, testdata; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStreamingSortInt(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ich := make(chan syncsort.Item[int], 10)

	testdata := make([]int, 1000)
	for i := range testdata {
		testdata[i] = i
	}

	seq := syncsort.NewSequencer(ctx, ich)

	items := make([]syncsort.Item[int], len(testdata))
	for i, v := range testdata {
		item, err := seq.NextItem(ctx, v)
		if err != nil {
			t.Fatalf("NextItem failed: %v", err)
		}
		items[i] = item
	}

	shuffled := shuffle(items)
	flattened := make([]int, len(shuffled))
	for i, s := range shuffled {
		flattened[i] = s.V
	}

	if got, want := flattened, testdata; reflect.DeepEqual(got, want) {
		t.Errorf("shuffle failed")
	}
	go send(shuffled, ich)

	reordered := []int{}
	for seq.Scan() {
		item := seq.Item()
		reordered = append(reordered, item)
	}
	if err := seq.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := reordered, testdata; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestStreamingCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ich := make(chan syncsort.Item[int], 10)

	seq := syncsort.NewSequencer(ctx, ich)

	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			item, err := seq.NextItem(ctx, 100)
			if err != nil {
				return
			}
			select {
			case ich <- item:
			case <-seq.Done():
				return
			}
		}
	})

	i := 0
	rnd := mrand.New(mrand.NewSource(time.Now().UnixNano())) // #nosec: G404
	stop := rnd.Intn(64)

	for seq.Scan() {
		if got := seq.Item(); got != 100 {
			t.Fatalf("Scan yielded unexpected item %d (expected 100)", got)
		}
		if i == stop {
			cancel()
		}
		i++
	}

	err := seq.Err()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("missing or incorrect error: %v", err)
	}
	wg.Wait()
}

func TestStreamingCancelNoDrain(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx, cancel := context.WithCancel(context.Background())
	ich := make(chan syncsort.Item[int])

	seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(4))

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 1; ; i++ {
			item, err := seq.NextItem(ctx, i)
			if err != nil {
				return
			}
			select {
			case ich <- item:
			case <-seq.Done():
				return
			}
		}
	})

	if !seq.Scan() || seq.Item() != 1 {
		t.Fatalf("expected first item 1, got %d (ok err=%v)", seq.Item(), seq.Err())
	}
	// Cancel and stop scanning immediately without draining remaining items.
	cancel()
	wg.Wait()
}

func TestStreamingEarlyStop(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ich := make(chan syncsort.Item[int])

	seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(3))
	defer seq.Stop()

	var wg sync.WaitGroup
	var prodErr error
	wg.Go(func() {
		for i := 1; i <= 1000; i++ {
			item, err := seq.NextItem(ctx, i)
			if err != nil {
				prodErr = err
				return
			}
			select {
			case ich <- item:
			case <-seq.Done():
				return
			}
		}
		close(ich)
	})

	var got []int
	for seq.Scan() {
		got = append(got, seq.Item())
		if len(got) == 5 {
			seq.Stop()
			break
		}
	}
	if seq.Scan() {
		t.Fatalf("expected Scan to return false after Stop")
	}
	if err := seq.Err(); err != nil {
		t.Fatalf("unexpected Err after Stop: %v", err)
	}
	wg.Wait()
	if prodErr != nil && !errors.Is(prodErr, syncsort.ErrStopped) {
		t.Fatalf("unexpected producer error: %v", prodErr)
	}
	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func sendItem[T any](ch chan<- syncsort.Item[T], done <-chan struct{}, item syncsort.Item[T]) bool {
	select {
	case ch <- item:
		return true
	case <-done:
		return false
	}
}

func sendReversed[T any](ch chan<- syncsort.Item[T], done <-chan struct{}, items []syncsort.Item[T]) bool {
	for j := len(items) - 1; j >= 0; j-- {
		if !sendItem(ch, done, items[j]) {
			return false
		}
	}
	return true
}

func sendOddEvenWorstCase[T any](ch chan<- syncsort.Item[T], done <-chan struct{}, items []syncsort.Item[T]) bool {
	for i := 1; i < len(items); i += 2 {
		if !sendItem(ch, done, items[i]) {
			return false
		}
	}
	lastEven := len(items) - 1
	if lastEven%2 != 0 {
		lastEven--
	}
	for i := lastEven; i >= 0; i -= 2 {
		if !sendItem(ch, done, items[i]) {
			return false
		}
	}
	return true
}

func verifySequentialInts(t *testing.T, seq *syncsort.Sequencer[int], total int) {
	t.Helper()
	var got []int
	for seq.Scan() {
		got = append(got, seq.Item())
	}
	if err := seq.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != total {
		t.Fatalf("got %d items, want %d", len(got), total)
	}
	for i, v := range got {
		if v != i+1 {
			t.Fatalf("at index %d: got %d, want %d", i, v, i+1)
		}
	}
}

func TestStreamingBoundedBackpressure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		const limit = 4
		const total = 20

		ich := make(chan syncsort.Item[int])
		seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(limit))
		defer seq.Stop()

		var issued atomic.Int64
		var wg sync.WaitGroup
		wg.Go(func() {
			defer close(ich)
			batch := make([]syncsort.Item[int], 0, limit)
			for i := range total {
				if i == limit {
					if !sendReversed(ich, seq.Done(), batch) {
						return
					}
				}
				item, err := seq.NextItem(ctx, i+1)
				if err != nil {
					return
				}
				issued.Add(1)
				if i < limit {
					batch = append(batch, item)
					continue
				}
				if !sendItem(ich, seq.Done(), item) {
					return
				}
			}
		})

		// Wait until the producer is blocked trying to issue item limit+1.
		synctest.Wait()
		if got := issued.Load(); got != limit {
			t.Fatalf("issued %d items while consumer hadn't scanned, want %d", got, limit)
		}

		verifySequentialInts(t, seq, total)
		wg.Wait()
	})
}

func produceWorstCaseBatches(ctx context.Context, seq *syncsort.Sequencer[int], ich chan<- syncsort.Item[int], total, limit int) {
	defer close(ich)
	pending := make([]syncsort.Item[int], 0, limit)
	for i := range total {
		item, err := seq.NextItem(ctx, i+1)
		if err != nil {
			return
		}
		pending = append(pending, item)
		if len(pending) == limit {
			if !sendOddEvenWorstCase(ich, seq.Done(), pending) {
				return
			}
			pending = pending[:0]
		}
	}
	sendOddEvenWorstCase(ich, seq.Done(), pending)
}

func TestStreamingBoundedWorstCaseOrdering(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	for _, limit := range []int{1, 2, 8, 16} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			ctx := context.Background()
			const total = 128

			ich := make(chan syncsort.Item[int])
			seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(limit))
			defer seq.Stop()

			go produceWorstCaseBatches(ctx, seq, ich, total, limit)
			verifySequentialInts(t, seq, total)
		})
	}
}

func TestStreamingUnboundedEvensThenOddsReversed(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	const total = 256

	ich := make(chan syncsort.Item[int])
	seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(0))
	defer seq.Stop()

	items := make([]syncsort.Item[int], total)
	for i := range total {
		item, err := seq.NextItem(ctx, i+1)
		if err != nil {
			t.Fatalf("NextItem failed: %v", err)
		}
		items[i] = item
	}

	go func() {
		defer close(ich)
		sendOddEvenWorstCase(ich, seq.Done(), items)
	}()

	verifySequentialInts(t, seq, total)
}

func TestStreamingBoundedAsyncEvensBeforeReversedOdds(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	for _, limit := range []int{1, 2, 8, 16} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			ctx := context.Background()
			const total = 128

			ich := make(chan syncsort.Item[int])
			seq := syncsort.NewSequencer(ctx, ich, syncsort.WithMaxBuffered(limit))
			defer seq.Stop()

			go func() {
				defer close(ich)
				var wg sync.WaitGroup
				for i := range total {
					item, err := seq.NextItem(ctx, i+1)
					if err != nil {
						return
					}
					if (i+1)%2 == 0 {
						if !sendItem(ich, seq.Done(), item) {
							return
						}
						continue
					}
					// Delay in-flight odds in reverse order within the active window so
					// evens race ahead and odds arrive out of order across limit boundaries.
					delay := time.Duration(limit-(i%limit)) * 50 * time.Microsecond
					wg.Go(func() {
						timer := time.NewTimer(delay)
						defer timer.Stop()
						select {
						case <-timer.C:
							sendItem(ich, seq.Done(), item)
						case <-seq.Done():
						}
					})
				}
				wg.Wait()
			}()

			verifySequentialInts(t, seq, total)
		})
	}
}

func TestStreamingIncompleteInputClosed(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ich := make(chan syncsort.Item[int], 4)
	seq := syncsort.NewSequencer(ctx, ich)
	defer seq.Stop()

	item1, err := seq.NextItem(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seq.NextItem(ctx, 2); err != nil { // allocate seq 2 but omit sending it
		t.Fatal(err)
	}
	item3, err := seq.NextItem(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	ich <- item1
	ich <- item3
	close(ich)

	var got []int
	for seq.Scan() {
		got = append(got, seq.Item())
	}
	if !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("got %v, want [1]", got)
	}
	if seq.Err() == nil {
		t.Fatalf("expected error when input channel closes with buffered gap")
	}
}
