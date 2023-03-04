// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package circular provides 'circular' data structures,
package circular

import (
	"reflect"
	"runtime"
	"testing"
)

func arange(s, n int) []int {
	if n == 0 {
		return nil
	}
	r := make([]int, n)
	for i := range r {
		r[i] = s + i
	}
	return r
}

func invariants[T any](t *testing.T, b *Buffer[T], head, tail, used, size int) {
	_, _, line, _ := runtime.Caller(1)
	if got, want := b.used, used; got != want {
		t.Errorf("line %v: used: got %v, want %v", line, got, want)
	}
	if got, want := b.head, head; got != want {
		t.Errorf("line %v: head: got %v, want %v", line, got, want)
	}
	if got, want := b.tail, tail; got != want {
		t.Errorf("line %v: tail: got %v, want %v", line, got, want)
	}
	if got, want := b.Cap(), size; got != want {
		t.Errorf("line %v: cap: got %v, want %v", line, got, want)
	}
}

func head[T any](t *testing.T, b *Buffer[T], n int, val []T) {
	_, _, line, _ := runtime.Caller(1)
	if got, want := b.Head(n), val; !reflect.DeepEqual(got, want) {
		t.Logf("%#v\n", b)
		t.Errorf("line %v: got %v, want %v", line, got, want)
	}
}

func testBufferLoop(t *testing.T, allowGrowth bool) {
	// Wrap around, chasing our tail, but allowing buffer growth.
	for bsize := 3; bsize <= 20; bsize++ {
		headVal := 1000
		tailVal := headVal
		blen := 0
		b := NewBuffer[int](bsize)
		bcap := bsize
		headIdx := 0
		for toAdd := 1; toAdd < bsize; toAdd++ {
			if !allowGrowth && b.Len()+toAdd+1 >= bsize {
				break
			}
			b.Append(arange(tailVal, toAdd+1)) // Add toAdd+1 elements
			tailVal += toAdd + 1
			blen += toAdd + 1
			if blen > bsize {
				bcap = blen
				headIdx = 0
			}
			for o := 0; o < bsize; o++ {
				// Remove and then add back toAdd elements
				head(t, b, toAdd, arange(headVal, toAdd))
				headIdx = (headIdx + toAdd) % b.Cap()
				invariants(t, b, headIdx, (headIdx+blen-toAdd-1)%b.Cap(), blen-toAdd, bcap)
				headVal += toAdd
				b.Append(arange(tailVal, toAdd))
				tailVal += toAdd
			}
			invariants(t, b, headIdx, (headIdx+blen-1)%b.Cap(), blen, bcap)
		}
		invariants(t, b, headIdx, (headIdx+blen-1)%b.Cap(), blen, bcap)
	}

}

func TestCircular(t *testing.T) {

	// Smallets buffer has a size of 1.
	b := NewBuffer[int](0)
	invariants(t, b, 0, 0, 0, 1)
	b = NewBuffer[int](1)
	invariants(t, b, 0, 0, 0, 1)

	bsize := 7
	b = NewBuffer[int](bsize)

	// Empty.
	invariants(t, b, 0, 0, 0, bsize)
	head(t, b, 0, arange(0, 0))
	invariants(t, b, 0, 0, 0, bsize)
	head(t, b, 10, arange(0, 0))

	// Fill and empty.
	b.Append(arange(0, bsize))
	invariants(t, b, 0, 6, 7, bsize)
	head(t, b, 7, arange(0, bsize))
	invariants(t, b, 7, 6, 0, bsize) // empty, so head/tail are not constrained
	b.Append(arange(10, bsize))
	invariants(t, b, 0, 6, 7, bsize)
	head(t, b, 7, arange(10, bsize))
	invariants(t, b, 7, 6, 0, bsize)

	// Append to limit, but no growth.
	b.Append(arange(100, 3))
	b.Append(arange(103, 4))
	invariants(t, b, 0, 6, 7, bsize)
	head(t, b, 7, arange(100, bsize))
	invariants(t, b, 7, 6, 0, bsize) // empty, so head/tail are not constrained

	// Append with growth.
	b.Append(arange(200, 100))
	invariants(t, b, 0, 99, 100, 100)
	head(t, b, 50, arange(200, 50))
	invariants(t, b, 50, 99, 50, 100)
	head(t, b, 500, arange(250, 50))

	testBufferLoop(t, false)
	testBufferLoop(t, true)

	// Test compaction.
	b.Compact()
	invariants(t, b, 0, 0, 0, 1)
	b.Append(arange(0, 10))
	invariants(t, b, 0, 9, 10, 10)
	b.Compact()
	invariants(t, b, 0, 9, 10, 10)
	head(t, b, 10, arange(0, 10))
	b.Compact()
	invariants(t, b, 0, 0, 0, 1)

	b.Append(arange(100, 10))
	head(t, b, 8, arange(100, 8))
	b.Append(arange(110, 4))
	invariants(t, b, 8, 3, 6, 10)
	b.Compact()
	invariants(t, b, 0, 5, 6, 6)
	head(t, b, 6, arange(108, 6))
}
