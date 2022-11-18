// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package codec provides support for interpreting byte slices as slices of
// other basic types such as runes, int64's or strings.
package codec

// Decoder represents the ability to decode a byte slice into a slice of
// some other data type.
type Decoder[T any] interface {
	Decode(input []byte) []T
}

type options struct {
	resizePrecent int
	sizePercent   int
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

// SizePercent requests that the initially allocated slice be 'percent' as
// large as the original input slice's size in bytes. A percent of 25 will
// divide the original size by 4 for example.
func SizePercent(percent int) Option {
	return func(o *options) {
		o.sizePercent = percent
	}
}

// NewDecode returns an instance of Decoder appropriate for the supplied
// function.
func NewDecoder[T any](fn func([]byte) (T, int), opts ...Option) Decoder[T] {
	dec := &decoder[T]{fn: fn}
	dec.resizePrecent = 100
	for _, fn := range opts {
		fn(&dec.options)
	}
	if dec.sizePercent == 0 {
		dec.sizePercent = 100
	}
	return dec
}

type decoder[T any] struct {
	options
	fn func([]byte) (T, int)
}

// Decode implements Decoder.
func (d *decoder[T]) Decode(input []byte) []T {
	if len(input) == 0 {
		return []T{}
	}
	out := make([]T, 0, len(input)/(100/d.sizePercent))
	cursor, i := 0, 0
	for {
		item, n := d.fn(input[cursor:])
		if n == 0 {
			break
		}
		out = append(out, item)
		i++
		cursor += n
		if cursor >= len(input) {
			break
		}
	}
	return resize(out[:i], d.resizePrecent)
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
//
//	(cap(slice) - len(slice)) / len(slice))
//
// exceeds the specified percentage.
func resize[T any](slice []T, percent int) []T {
	if resizedNeeded(len(slice), cap(slice), percent) {
		r := make([]T, len(slice))
		copy(r, slice)
		return r
	}
	return slice
}
