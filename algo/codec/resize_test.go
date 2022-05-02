// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package codec

import (
	"reflect"
	"strings"
	"testing"
)

func testResize[T any](t *testing.T, input []T, percent, slen, scap int, same bool) {
	resized := resize(input, percent)
	if got, want := len(resized), slen; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(resized), scap; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := reflect.ValueOf(input).Pointer() == reflect.ValueOf(resized).Pointer(), same; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResize(t *testing.T) {
	testResize(t, make([]uint8, 10, 20), 100, 10, 20, true)
	testResize(t, make([]uint8, 10, 21), 100, 10, 10, false)
	testResize(t, make([]int32, 3), 100, 3, 3, true)
	testResize(t, make([]int32, 3, 7), 100, 3, 3, false)
	testResize(t, make([]string, 3, 4), 10, 3, 3, false)
	testResize(t, make([]int64, 3, 4), 10, 3, 3, false)
	testResize(t, make([]int64, 10, 15), 50, 10, 15, true)
	testResize(t, make([]int64, 10, 15), 49, 10, 10, false)

	waster := func(buf []byte) (string, int) {
		return "A", 10
	}

	dec := NewDecoder(waster)
	used := dec.Decode([]byte(strings.Repeat("B", 100)))
	if got, want := len(used), 10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	dec = NewDecoder(waster, ResizePercent(10000))
	used = dec.Decode([]byte(strings.Repeat("B", 100)))
	if got, want := len(used), 10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 100; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestEarlyExit(t *testing.T) {
	earlyExit := func(buf []byte) (string, int) {
		return "", 0
	}
	used := NewDecoder(earlyExit).Decode([]byte("BBCXXX"))
	if got, want := len(used), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	used = NewDecoder(earlyExit, SizePercent(25)).Decode([]byte("BBCXXX"))
	if got, want := len(used), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
