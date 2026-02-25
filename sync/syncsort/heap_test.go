// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package syncsort_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand" // #nosec: G404
	"reflect"
	"sync"
	"testing"
	"time"

	"cloudeng.io/sync/syncsort"
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

func partitionSlices[T any](seq *syncsort.Sequencer[[]T], data []T, blockSize int) []syncsort.Item[[]T] {
	dataSize := len(data)
	numBlocks := (dataSize / blockSize) + 1
	sblocks := make([]syncsort.Item[[]T], numBlocks)
	for i := range numBlocks {
		size := blockSize
		offset := i * size
		if offset+size > dataSize {
			size = dataSize - offset
		}
		sblocks[i] = seq.NextItem(data[offset : offset+size])
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
	ctx := context.Background()
	ich := make(chan syncsort.Item[[]byte], 10)
	testdata := generateRandomBytes(4096)

	seq := syncsort.NewSequencer(ctx, ich)
	blocks := partitionSlices(seq, testdata, 33)

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
	ctx := context.Background()
	ich := make(chan syncsort.Item[int], 10)

	testdata := make([]int, 1000)
	for i := range testdata {
		testdata[i] = i
	}

	seq := syncsort.NewSequencer(ctx, ich)

	items := make([]syncsort.Item[int], len(testdata))
	for i, v := range testdata {
		items[i] = seq.NextItem(v)
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
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ich := make(chan syncsort.Item[int], 10)

	seq := syncsort.NewSequencer(ctx, ich)

	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case ich <- seq.NextItem(100):
			case <-ctx.Done():
				return
			}
		}
	})

	i := 0

	rnd := mrand.New(mrand.NewSource(time.Now().UnixNano())) // #nosec: G404
	stop := rnd.Intn(64)

	for seq.Scan() {
		if i == stop {
			cancel()
		}
		i++
	}

	err := seq.Err()
	if err == nil || err.Error() != "context canceled" {
		t.Fatalf("missing or incorrect error: %v", err)
	}
	wg.Wait()
}
