// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package codec

import (
	"reflect"
	"strings"
	"testing"
)

func TestResize(t *testing.T) {
	for i, tc := range []struct {
		input             interface{}
		percent, len, cap int
		same              bool
	}{
		{make([]uint8, 10, 20), 100, 10, 20, true},
		{make([]uint8, 10, 21), 100, 10, 10, false},
		{make([]int32, 3, 3), 100, 3, 3, true},
		{make([]int32, 3, 7), 100, 3, 3, false},
		{make([]string, 3, 4), 10, 3, 3, false},
		{make([]int64, 3, 4), 10, 3, 3, false},
		{make([]int64, 10, 15), 50, 10, 15, true},
		{make([]int64, 10, 15), 49, 10, 10, false},
	} {
		resized := resize(tc.input, tc.percent)
		if got, want := reflect.ValueOf(resized).Len(), tc.len; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := reflect.ValueOf(resized).Cap(), tc.cap; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := reflect.ValueOf(tc.input).Pointer() == reflect.ValueOf(resized).Pointer(), tc.same; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	waster := func(buf []byte) (string, int) {
		return "A", 10
	}

	dec, _ := NewDecoder(waster)
	used := dec.Decode([]byte(strings.Repeat("B", 100))).([]string)
	if got, want := len(used), 10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	dec, _ = NewDecoder(waster, ResizePercent(10000))
	used = dec.Decode([]byte(strings.Repeat("B", 100))).([]string)
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
	dec, _ := NewDecoder(earlyExit)
	used := dec.Decode([]byte("BBCXXX")).([]string)
	if got, want := len(used), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(used), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
