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

func TestDecoder(t *testing.T) {
	for i, tc := range []struct {
		fn        interface{}
		input     []byte
		output    interface{}
		outputLen int
	}{
		{utf8.DecodeRune, []byte(""), "", 0},
		{utf8.DecodeRune, []byte("日本語"), "日本語", 3},
		{func(input []byte) (byte, int) { return input[0], 1 }, []byte("日本語"), []byte("日本語"), 9},
		{hash64Lines, []byte("AA\nBB"), []int64{650879030918179831, 653890593267282085}, 2},
		{stringLines, []byte("AA\nBB"), []string{"AA", "BB"}, 2},
	} {
		dec, err := codec.NewDecoder(tc.fn)
		if err != nil {
			t.Errorf("%v: NewDecoder for %T: %v", i, tc.fn, err)
		}
		output := dec.Decode(tc.input)
		var ni int
		switch v := output.(type) {
		case []int64:
			ni = len(v)
			if got, want := v, tc.output.([]int64); !reflect.DeepEqual(got, want) {
				t.Errorf("%v: got %v, want %v", i, got, want)
			}
		case []int32:
			ni = len(v)
			if got, want := string(v), tc.output; !reflect.DeepEqual(got, want) {
				t.Errorf("%v: got %v, want %v", i, got, want)
			}
		case []byte:
			ni = len(v)
			if got, want := output, tc.output; !reflect.DeepEqual(got, want) {
				t.Errorf("%v: got %v, want %v", i, got, want)
			}
		case []string:
			ni = len(v)
			if got, want := output, tc.output.([]string); !reflect.DeepEqual(got, want) {
				t.Errorf("%v: got %v, want %v", i, got, want)
			}
		default:
			t.Fatalf("unsupported type: %T", output)
		}
		if got, want := ni, tc.outputLen; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

}

func ExampleDecoder() {
	runeDecoder, _ := codec.NewDecoder(utf8.DecodeRune)
	decoded := runeDecoder.Decode([]byte("日本語"))
	fmt.Println(len(decoded.([]int32)))
	// Output:
	// 3
}
