// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package codec provides support for interpreting byte slices as slices of
// other basic types such as runes, int64's or strings. Go's lack of generics
// make this awkward and this package currently supports a fixed set of
// basic types (slices of byte/uint8, rune/int32, int64 and string).
package codec

import "fmt"

// Decoder represents the ability to decode a byte slice into a slice of
// some other data type.
type Decoder interface {
	Decode(input []byte) interface{}
}

type options struct {
	resizePrecent int
}

// Option represents an option accepted by NewDecoder.
type Option func(*options)

// ResizePercent requests that the returned slice be reallocated if the
// ratio of unused to used capacity exceeds the specified percentage.
// That is, if cap(slice) - len(slice)) / len(slice) exceeds the percentage
// new underlying storage is allocated and contents copied. The default
// value for ResizePercent is 100.
func ResizePercent(percent int) Option {
	return func(o *options) {
		o.resizePrecent = percent
	}
}

// NewDecode returns an instance of Decoder appropriate for the supplied
// function. The currently supported function signatures are:
//   func([]byte) (uint8, int)
//   func([]byte) (int32, int)
//   func([]byte) (int64, int)
//   func([]byte) (string, int)
func NewDecoder(fn interface{}, opts ...Option) (Decoder, error) {
	var o options
	o.resizePrecent = 100
	for _, fn := range opts {
		fn(&o)
	}
	switch v := fn.(type) {
	case func([]byte) (uint8, int):
		return &decoder8{o, v}, nil
	case func([]byte) (int32, int):
		return &decoder32{o, v}, nil
	case func([]byte) (int64, int):
		return &decoder64{o, v}, nil
	case func([]byte) (string, int):
		return &decoderString{o, v}, nil
	}
	return nil, fmt.Errorf("unsupported type for decoder function: %T", fn)
}

type decoder8 struct {
	options
	fn func([]byte) (uint8, int)
}

// Decode implements Decoder.
func (d *decoder8) Decode(input []byte) interface{} {
	out := make([]uint8, len(input))
	n := decode(input, func(in []byte, i int) (n int) {
		out[i], n = d.fn(in)
		return
	})
	return resize(out[:n], d.resizePrecent)
}

type decoder32 struct {
	options
	fn func([]byte) (int32, int)
}

// Decode implements Decoder.
func (d *decoder32) Decode(input []byte) interface{} {
	out := make([]int32, len(input))
	n := decode(input, func(in []byte, i int) (n int) {
		out[i], n = d.fn(in)
		return
	})
	return resize(out[:n], d.resizePrecent)
}

type decoder64 struct {
	options
	fn func([]byte) (int64, int)
}

// Decode implements Decoder.
func (d *decoder64) Decode(input []byte) interface{} {
	out := make([]int64, len(input))
	n := decode(input, func(in []byte, i int) (n int) {
		out[i], n = d.fn(in)
		return
	})
	return resize(out[:n], d.resizePrecent)
}

type decoderString struct {
	options
	fn func([]byte) (string, int)
}

// Decode implements Decoder.
func (d *decoderString) Decode(input []byte) interface{} {
	out := make([]string, len(input))
	n := decode(input, func(in []byte, i int) (n int) {
		out[i], n = d.fn(in)
		return
	})
	return resize(out[:n], d.resizePrecent)
}

func decode(input []byte, fn func([]byte, int) int) int {
	if len(input) == 0 {
		return 0
	}
	cursor, i := 0, 0
	for {
		n := fn(input[cursor:], i)
		if n == 0 {
			break
		}
		i++
		cursor += n
		if cursor >= len(input) {
			break
		}
	}
	return i
}

func resizedNeeded(used, available int, percent int) bool {
	wasted := available - used
	if used == 0 {
		used = 1
	}
	return ((wasted * 100) / used) > percent
}

// resize will allocate new underlying storage and copy the contents of
// slice to it if the ratio of wasted to used, ie:
//   (cap(slice) - len(slice)) / len(slice))
// exceeds the specified percentage.
func resize(slice interface{}, percent int) interface{} {
	switch v := slice.(type) {
	case []uint8:
		if resizedNeeded(len(v), cap(v), percent) {
			r := make([]uint8, len(v))
			copy(r, v)
			return r
		}
	case []int32:
		if resizedNeeded(len(v), cap(v), percent) {
			r := make([]int32, len(v))
			copy(r, v)
			return r
		}
	case []int64:
		if resizedNeeded(len(v), cap(v), percent) {
			r := make([]int64, len(v))
			copy(r, v)
			return r
		}
	case []string:
		if resizedNeeded(len(v), cap(v), percent) {
			r := make([]string, len(v))
			copy(r, v)
			return r
		}
	default:
		panic(fmt.Sprintf("unsupported type %T", slice))
	}
	return slice
}
