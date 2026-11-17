// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
)

// Edge says which end of the span that a value names is wanted. A value
// names all of the span to its precision, so the date 2024-03-05 is the whole
// of that day: its Lower edge is the start of the day and its Upper edge the
// start of the next. Values with no coarser precision, such as an int, have a
// span of one, so the Upper edge of 5 is 6.
type Edge int

const (
	Lower Edge = iota
	Upper
)

// Codec defines how a type is parsed and compared by Bound and Interval.
// It is a type parameter, rather than a value, so that a Bound or Interval has
// a usable zero value and so can be a field of a struct whose flags are
// registered by RegisterFlagsInStruct. Implementations are therefore empty
// struct types, such as TimeCodec.
type Codec[T any] interface {
	// Parse parses s, returning the edge of the span that it names.
	Parse(s string, edge Edge) (T, error)
	// Compare returns -1, 0 or +1 as a is before, equal to or after b.
	Compare(a, b T) int
}

// Bound is a flag.Value for a single value of type T that is compared with
// others, such as --since or --max-size. It names a span, see Edge, and so a
// time or date can be asked about as a whole. The zero value is unset.
//
// A Bound is encoded to and from text, such as for JSON and YAML, as the text
// it was set from.
type Bound[T any, C Codec[T]] struct {
	text         string
	lower, upper T
	set          bool
}

// Set implements flag.Value.
func (b *Bound[T, C]) Set(s string) error {
	var c C
	lower, err := c.Parse(s, Lower)
	if err != nil {
		return err
	}
	upper, err := c.Parse(s, Upper)
	if err != nil {
		return err
	}
	*b = Bound[T, C]{text: s, lower: lower, upper: upper, set: true}
	return nil
}

// String implements flag.Value, returning the value as it was set.
func (b *Bound[T, C]) String() string {
	if b == nil {
		return ""
	}
	return b.text
}

// Get implements flag.Getter, returning the lower edge.
func (b *Bound[T, C]) Get() any { return b.lower }

// IsDefault returns true if the value has not been set.
func (b *Bound[T, C]) IsDefault() bool { return !b.set }

// Lower returns the first value that the bound names.
func (b *Bound[T, C]) Lower() T { return b.lower }

// Upper returns the value just after the last that the bound names.
func (b *Bound[T, C]) Upper() T { return b.upper }

// Contains returns true if v is within the span that the bound names.
// It is false if the bound is not set.
func (b *Bound[T, C]) Contains(v T) bool {
	var c C
	return b.set && c.Compare(v, b.lower) >= 0 && c.Compare(v, b.upper) < 0
}

// Before returns true if the whole of the span that the bound names is before
// v, ie. if v is at or after its Upper edge. It is false if the bound is not
// set.
func (b *Bound[T, C]) Before(v T) bool {
	var c C
	return b.set && c.Compare(v, b.upper) >= 0
}

// After returns true if the whole of the span that the bound names is after
// v, ie. if v is before its Lower edge. It is false if the bound is not set.
func (b *Bound[T, C]) After(v T) bool {
	var c C
	return b.set && c.Compare(v, b.lower) < 0
}

// MarshalText implements encoding.TextMarshaler.
func (b Bound[T, C]) MarshalText() ([]byte, error) { return []byte(b.text), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *Bound[T, C]) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*b = Bound[T, C]{}
		return nil
	}
	return b.Set(string(text))
}

// IntervalSeparator separates the two ends of an Interval.
const IntervalSeparator = ".."

// Interval is a flag.Value for a range of values of type T, [From, To), that
// either end of which may be left open. The zero value is unset.
//
// It is written <from>..<to>, <from>.. or ..<to>, or as a single value:
//
//	a..b  : from the start of a to the end of b, so 2024-03-01..2024-03-05 includes all of the fifth
//	a..   : from the start of a onwards
//	..b   : up to the end of b
//	a     : all of a, so 2024-03 is the month of March and 5 is exactly 5
//
// The separator is not one that any of the values that it is used with
// contain, unlike the '-' of RangeSpec which dates and negative numbers
// do. An Interval that is empty or ends before it starts is an error.
//
// An Interval is encoded to and from text, such as for JSON and YAML, as the
// text it was set from.
type Interval[T any, C Codec[T]] struct {
	text           string
	from, to       T
	hasFrom, hasTo bool
	set            bool
}

// Set implements flag.Value.
func (iv *Interval[T, C]) Set(s string) error {
	var c C
	next := Interval[T, C]{text: s, set: true}
	lhs, rhs, ranged := strings.Cut(s, IntervalSeparator)
	if !ranged {
		rhs = lhs
	}
	if lhs == "" && rhs == "" {
		return fmt.Errorf("invalid interval %q: it has no values", s)
	}
	var err error
	if lhs != "" {
		if next.from, err = c.Parse(lhs, Lower); err != nil {
			return fmt.Errorf("invalid interval %q: %w", s, err)
		}
		next.hasFrom = true
	}
	if rhs != "" {
		if next.to, err = c.Parse(rhs, Upper); err != nil {
			return fmt.Errorf("invalid interval %q: %w", s, err)
		}
		next.hasTo = true
	}
	if next.hasFrom && next.hasTo && c.Compare(next.to, next.from) <= 0 {
		return fmt.Errorf("invalid interval %q: it is empty or ends before it starts", s)
	}
	*iv = next
	return nil
}

// String implements flag.Value, returning the interval as it was set.
func (iv *Interval[T, C]) String() string {
	if iv == nil {
		return ""
	}
	return iv.text
}

// Get implements flag.Getter, returning the interval itself.
func (iv *Interval[T, C]) Get() any { return iv }

// IsDefault returns true if the value has not been set.
func (iv *Interval[T, C]) IsDefault() bool { return !iv.set }

// From returns the start of the interval, and false if it is open.
func (iv *Interval[T, C]) From() (T, bool) { return iv.from, iv.hasFrom }

// To returns the end of the interval, which is not within it, and false if it
// is open.
func (iv *Interval[T, C]) To() (T, bool) { return iv.to, iv.hasTo }

// Contains returns true if v is within the interval. It is false if the
// interval is not set.
func (iv *Interval[T, C]) Contains(v T) bool {
	var c C
	if !iv.set {
		return false
	}
	return (!iv.hasFrom || c.Compare(v, iv.from) >= 0) && (!iv.hasTo || c.Compare(v, iv.to) < 0)
}

// Overlaps returns true if the intervals have any value in common. It is
// false if either is not set.
func (iv *Interval[T, C]) Overlaps(other *Interval[T, C]) bool {
	var c C
	if !iv.set || !other.set {
		return false
	}
	before := func(a Interval[T, C], b *Interval[T, C]) bool { // a lies entirely before b
		return a.hasTo && b.hasFrom && c.Compare(a.to, b.from) <= 0
	}
	return !before(*iv, other) && !before(*other, iv)
}

// MarshalText implements encoding.TextMarshaler.
func (iv Interval[T, C]) MarshalText() ([]byte, error) { return []byte(iv.text), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (iv *Interval[T, C]) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*iv = Interval[T, C]{}
		return nil
	}
	return iv.Set(string(text))
}

// TimeCodec is the Codec for time.Time, using cmdtypes.DefaultTimeParser: a
// date and time, a date, month or year, or a time relative to now such as
// -7d, see cmdtypes.TimeSpanFormats.
type TimeCodec struct{}

// Parse implements Codec.
func (TimeCodec) Parse(s string, edge Edge) (time.Time, error) {
	start, end, err := cmdtypes.DefaultTimeParser.Span(s)
	if edge == Upper {
		return end, err
	}
	return start, err
}

// Compare implements Codec.
func (TimeCodec) Compare(a, b time.Time) int { return a.Compare(b) }

// IntCodec is the Codec for int.
type IntCodec struct{}

// Parse implements Codec.
func (IntCodec) Parse(s string, edge Edge) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid int %q", s)
	}
	if edge == Upper {
		n++
	}
	return n, nil
}

// Compare implements Codec.
func (IntCodec) Compare(a, b int) int { return cmpInt(int64(a), int64(b)) }

// ByteSizeCodec is the Codec for cmdtypes.ByteSize, accepting the same
// notations as it.
type ByteSizeCodec struct{}

// Parse implements Codec.
func (ByteSizeCodec) Parse(s string, edge Edge) (cmdtypes.ByteSize, error) {
	n, err := cmdtypes.ParseByteSize(s)
	if edge == Upper {
		n++
	}
	return n, err
}

// Compare implements Codec.
func (ByteSizeCodec) Compare(a, b cmdtypes.ByteSize) int { return cmpInt(int64(a), int64(b)) }

// DurationCodec is the Codec for time.Duration, accepting the notations of
// cmdtypes.ParseDuration, with a leading '+' or '-' allowed.
type DurationCodec struct{}

// Parse implements Codec.
func (DurationCodec) Parse(s string, edge Edge) (time.Duration, error) {
	t := strings.TrimSpace(s)
	neg := strings.HasPrefix(t, "-")
	d, err := cmdtypes.ParseDuration(strings.TrimLeft(t, "+-"))
	if err != nil {
		return 0, err
	}
	if neg {
		d = -d
	}
	if edge == Upper {
		d++
	}
	return d, nil
}

// Compare implements Codec.
func (DurationCodec) Compare(a, b time.Duration) int { return cmpInt(int64(a), int64(b)) }

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

type (
	// TimeBound is a Bound of a time, see TimeCodec.
	TimeBound = Bound[time.Time, TimeCodec]
	// IntBound is a Bound of an int.
	IntBound = Bound[int, IntCodec]
	// ByteSizeBound is a Bound of a number of bytes.
	ByteSizeBound = Bound[cmdtypes.ByteSize, ByteSizeCodec]
	// DurationBound is a Bound of a duration.
	DurationBound = Bound[time.Duration, DurationCodec]

	// TimeInterval is an Interval of times, see TimeCodec.
	TimeInterval = Interval[time.Time, TimeCodec]
	// IntInterval is an Interval of ints.
	IntInterval = Interval[int, IntCodec]
	// ByteSizeInterval is an Interval of numbers of bytes.
	ByteSizeInterval = Interval[cmdtypes.ByteSize, ByteSizeCodec]
	// DurationInterval is an Interval of durations.
	DurationInterval = Interval[time.Duration, DurationCodec]
)
