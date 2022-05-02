// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package codec_test

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"reflect"
	"testing"
	"unicode/utf8"

	"cloudeng.io/algo/codec"
)

func hash64Lines(data []byte) (int64, int) {
	idx := bytes.Index(data, []byte{'\n'})
	if idx < 0 {
		idx = len(data)
	}
	h := fnv.New64a()
	h.Write(data[:idx])
	sum := h.Sum64()
	return int64(sum), idx + 1
}

func stringLines(data []byte) (string, int) {
	idx := bytes.Index(data, []byte{'\n'})
	if idx < 0 {
		idx = len(data)
	}
	return string(data[:idx]), idx + 1
}

func testDecode[T any](t *testing.T, fn func([]byte) (T, int), input []byte, output []T, outputLen int) {
	decoded := codec.NewDecoder(fn).Decode(input)
	if got, want := decoded, output; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(decoded), outputLen; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDecoder(t *testing.T) {

	testDecode(t, utf8.DecodeRune, []byte(""), []rune(""), 0)
	testDecode(t, utf8.DecodeRune, []byte("日本語"), []rune("日本語"), 3)
	testDecode(t, func(input []byte) (byte, int) { return input[0], 1 }, []byte("日本語"), []byte("日本語"), 9)
	testDecode(t, hash64Lines, []byte("AA\nBB"), []int64{650879030918179831, 653890593267282085}, 2)
	testDecode(t, stringLines, []byte("AA\nBB"), []string{"AA", "BB"}, 2)
}

func ExampleDecoder() {
	decoded := codec.NewDecoder(utf8.DecodeRune).Decode([]byte("日本語"))
	fmt.Println(len(decoded))
	// Output:
	// 3
}
