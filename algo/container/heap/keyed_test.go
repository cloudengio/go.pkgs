// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"cloudeng.io/algo/container/heap"
)

func assertNextKI64(t *testing.T, h *heap.KeyedInt64, ek string, ev int64) {
	k, v := h.Pop()
	_, _, line, _ := runtime.Caller(2)
	if got, want := k, ek; got != want {
		t.Errorf("line %v: got %v, want %v", line, got, want)
	}
	if got, want := v, ev; got != want {
		t.Errorf("line %v: got %v, want %v", line, got, want)
	}
}

func assertStatsKI64(t *testing.T, h *heap.KeyedInt64, l int, v int64) {
	_, _, line, _ := runtime.Caller(2)
	if got, want := h.Len(), l; got != want {
		t.Errorf("line %v: got %v, want %v", line, got, want)
	}
	if got, want := h.Total(), v; got != want {
		t.Errorf("line %v: got %v, want %v", line, got, want)
	}
}

func TestKeyedHeapSimple(t *testing.T) {
	var h *heap.KeyedInt64
	assertNext := func(k string, v int64) {
		assertNextKI64(t, h, k, v)
	}
	assertStats := func(l int, v int64) {
		assertStatsKI64(t, h, l, v)
	}
	h = heap.NewKeyedInt64(false)

	h.Update("a", 33)
	assertStats(1, 33)
	h.Update("b", 44)
	assertStats(2, 77)
	h.Update("c", 12)
	assertStats(3, 89)
	h.Update("c", 13)
	assertStats(3, 90)

	assertNext("c", 13)
	assertNext("a", 33)
	assertNext("b", 44)
	assertStats(0, 0)

	h.Update("z", 360)
	h.Update("x", 31323)
	h.Update("y", -2)
	assertStats(3, 31681)

	if err := isAscending(h.TopN(100)); err != nil {
		t.Fatal(err)
	}
	assertStats(0, 0)

	h = heap.NewKeyedInt64(true)
	h.Update("a", 33)
	assertStats(1, 33)
	h.Update("b", 44)
	h.Update("b", 45)
	assertStats(2, 78)
	h.Update("c", 12)
	assertStats(3, 90)

	assertNext("b", 45)
	assertNext("a", 33)
	assertNext("c", 12)
	assertStats(0, 0)

	h.Remove("nothere")
	assertStats(0, 0)
	h.Update("a", 22)
	assertStats(1, 22)
	h.Remove("a")
	assertStats(0, 0)
	h.Update("c", 21)
	assertStats(1, 21)
	h.Update("d", 1)
	assertStats(2, 22)

	assertNext("c", 21)
	assertStats(1, 1)
	assertNext("d", 1)
	assertStats(0, 0)

}

func fillrand(h *heap.KeyedInt64, n int) {
	for i := 0; i < n; i++ {
		v := rand.Int63()
		h.Update(strconv.Itoa(int(v)), v)
	}
}

func isAscending(items []struct {
	Key   string
	Value int64
}) error {
	prev := items[0]
	for i, kv := range items[1:] {
		if got, want := prev.Value, kv.Value; got > want {
			return fmt.Errorf("%v: key %v <= %v", i, got, want)
		}
		prev = kv
	}
	return nil
}

func TestKeyedHeap(t *testing.T) {
	h := heap.NewKeyedInt64(false)

	var wg sync.WaitGroup
	iter := 8
	perIter := 1000
	for i := 0; i < iter; i++ {
		wg.Add(1)
		go func() {
			fillrand(h, perIter)
			wg.Done()
		}()
	}
	wg.Wait()
	if got, want := h.Len(), iter*perIter; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	jsonBuf, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}

	gobBuf := &bytes.Buffer{}
	enc := gob.NewEncoder(gobBuf)
	if err := enc.Encode(h); err != nil {
		t.Fatal(err)
	}

	olen, ototal := h.Len(), h.Total()
	n := 100
	top := h.TopN(n)
	if err := isAscending(top); err != nil {
		t.Fatal(err)
	}

	jsonHeap := &heap.KeyedInt64{}
	if err := json.Unmarshal(jsonBuf, jsonHeap); err != nil {
		t.Fatal(err)
	}
	gobHeap := &heap.KeyedInt64{}
	dec := gob.NewDecoder(gobBuf)
	if err := dec.Decode(&gobHeap); err != nil {
		t.Fatal(err)
	}

	assertStats := func(l int, v int64) {
		assertStatsKI64(t, jsonHeap, l, v)
		assertStatsKI64(t, gobHeap, l, v)
	}

	assertStats(olen, ototal)
	if got, want := jsonHeap.TopN(n), top; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := gobHeap.TopN(n), top; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	assertStats(h.Len(), h.Total())
}
