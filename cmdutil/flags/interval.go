// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
)

// ErrInvalidValue is wrapped by the error returned for any value that cannot
// be parsed by Bound, Interval or Intervals, use errors.Is to test for it.
// The error also wraps the underlying cause, if there is one.
var ErrInvalidValue = errors.New("invalid value")

type parseError struct {
	msg   string
	cause error
}

func (e *parseError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return e.msg + ": " + e.cause.Error()
}

func (e *parseError) Unwrap() []error {
	if e.cause == nil {
		return []error{ErrInvalidValue}
	}
	return []error{ErrInvalidValue, e.cause}
}

func invalid(msg string, cause error) error {
	return &parseError{msg: msg, cause: cause}
}

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

type spanParser[T any] interface {
	parseSpan(s string) (lower, upper T, err error)
}

// endRelative is implemented by codecs whose values are positions that may
// be counted back from an end, written ^<offset>, where ^1 is the last item.
type endRelative[T any] interface {
	parseOffset(s string) (T, error)
	fromEnd(offset, end T, edge Edge) (T, error)
}

const relativePrefix = "^"

// isRelative returns true if s is relative to the end. The prefix is only
// special for codecs that support it, so a string may still start with ^.
func isRelative[T any, C Codec[T]](c C, s string) bool {
	_, ok := any(c).(endRelative[T])
	return ok && strings.HasPrefix(s, relativePrefix)
}

func parseEndpoint[T any, C Codec[T]](c C, s string, edge Edge) (v T, relative bool, err error) {
	if !isRelative(c, s) {
		v, err = c.Parse(s, edge)
		return v, false, err
	}
	v, err = any(c).(endRelative[T]).parseOffset(s[len(relativePrefix):])
	return v, true, err
}

type rangeParser[T any] interface {
	parseRange(lhs, rhs string) (from, to T, err error)
}

func parseBound[T any, C Codec[T]](c C, s string) (T, T, error) {
	if sp, ok := any(c).(spanParser[T]); ok {
		return sp.parseSpan(s)
	}
	lower, err := c.Parse(s, Lower)
	if err != nil {
		return lower, lower, err
	}
	upper, err := c.Parse(s, Upper)
	if err != nil {
		return lower, lower, err
	}
	return lower, upper, nil
}

// Set implements flag.Value.
func (b *Bound[T, C]) Set(s string) error {
	var c C
	lower, upper, err := parseBound(c, s)
	if err != nil {
		return invalid(fmt.Sprintf("invalid value %q", s), err)
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

// IntervalSeparator is the default separator between the two ends of an
// Interval, see Interval.SetSeparator.
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
// For types that are positions, such as IntInterval, a value may be counted
// back from the end with a ^ prefix, where ^1 is the last item, eg. ^3..^1 is
// the last three items, 2..^2 is all but the first two and last, and ^1 alone
// is the last. Such an interval must be resolved, see Resolve, before use.
//
// The default separator is not one that any of the values that it is used
// with contain, unlike '-' which dates and negative numbers do. Use
// SetSeparator to use another. An Interval that is empty or ends before it
// starts is an error, wrapping ErrInvalidValue.
//
// An Interval is encoded to and from text, such as for JSON and YAML, as the
// text it was set from.
type Interval[T any, C Codec[T]] struct {
	sep            string
	text           string
	from, to       T
	hasFrom, hasTo bool
	fromRel, toRel bool // from/to are offsets back from the end, see Resolve.
	set            bool
}

// SetSeparator sets the separator used by Set and UnmarshalText, which
// defaults to IntervalSeparator if sep is empty. It must not appear within
// the values being parsed, so '-' is a poor choice for times and negative
// numbers. It is kept when the interval is later set, and should be called
// before the interval is set, eg. after RegisterFlagsInStruct and before
// parsing the flags. A text value is always read with the separator
// currently in effect. A default value given in a struct tag is parsed when
// the flags are registered, so must use the default separator.
func (iv *Interval[T, C]) SetSeparator(sep string) { iv.sep = sep }

func (iv *Interval[T, C]) separator() string {
	if iv.sep == "" {
		return IntervalSeparator
	}
	return iv.sep
}

func parseIntervalEndpoints[T any, C Codec[T]](c C, s, lhs, rhs string) (next Interval[T, C], err error) {
	next = Interval[T, C]{text: s, set: true}
	rel := isRelative(c, lhs) || isRelative(c, rhs)
	if rp, ok := any(c).(rangeParser[T]); ok && lhs != "" && rhs != "" && !rel {
		next.from, next.to, err = rp.parseRange(lhs, rhs)
		next.hasFrom, next.hasTo = true, true
		return next, err
	}
	if lhs != "" {
		if next.from, next.fromRel, err = parseEndpoint(c, lhs, Lower); err != nil {
			return next, err
		}
		next.hasFrom = true
	}
	if rhs != "" {
		if next.to, next.toRel, err = parseEndpoint(c, rhs, Upper); err != nil {
			return next, err
		}
		next.hasTo = true
	}
	return next, nil
}

func (iv *Interval[T, C]) unresolved() bool { return iv.fromRel || iv.toRel }

// checkOrder returns an error if the interval is empty or ends before it
// starts, as far as can be determined before it is resolved.
func (iv *Interval[T, C]) checkOrder() error {
	var c C
	switch {
	case !iv.hasFrom || !iv.hasTo:
	case iv.fromRel && iv.toRel:
		// Offsets count back, so a larger one is earlier.
		if c.Compare(iv.to, iv.from) > 0 {
			return errors.New("it ends before it starts")
		}
	case iv.unresolved():
	case c.Compare(iv.to, iv.from) <= 0:
		return errors.New("it is empty or ends before it starts")
	}
	return nil
}

// Set implements flag.Value.
func (iv *Interval[T, C]) Set(s string) error {
	var c C
	errPrefix := fmt.Sprintf("invalid interval %q", s)
	lhs, rhs, ranged := strings.Cut(s, iv.separator())
	if !ranged {
		return iv.setSingle(c, s, errPrefix)
	}
	if lhs == "" && rhs == "" {
		return invalid(errPrefix+": it has no values", nil)
	}
	next, err := parseIntervalEndpoints(c, s, lhs, rhs)
	if err == nil {
		err = next.checkOrder()
	}
	if err != nil {
		return invalid(errPrefix, err)
	}
	next.sep = iv.sep
	*iv = next
	return nil
}

func (iv *Interval[T, C]) setSingle(c C, s, errPrefix string) error {
	if s == "" {
		return invalid(errPrefix+": it has no values", nil)
	}
	next := Interval[T, C]{sep: iv.sep, text: s, hasFrom: true, hasTo: true, set: true}
	if isRelative(c, s) {
		off, _, err := parseEndpoint(c, s, Lower)
		if err != nil {
			return invalid(errPrefix, err)
		}
		next.from, next.to, next.fromRel, next.toRel = off, off, true, true
	} else {
		from, to, err := parseBound(c, s)
		if err != nil {
			return invalid(errPrefix, err)
		}
		next.from, next.to = from, to
	}
	*iv = next
	return nil
}

// Resolve returns the interval with any values that are relative to the end
// replaced by absolute ones, where end is the exclusive end, such as the
// number of items. It is an error if the result is empty, ends before it
// starts or lies before the first item. An interval with no such values is
// returned unchanged.
func (iv *Interval[T, C]) Resolve(end T) (Interval[T, C], error) {
	var c C
	out := *iv
	er, ok := any(c).(endRelative[T])
	if !iv.set || !iv.unresolved() || !ok {
		return out, nil
	}
	errPrefix := fmt.Sprintf("invalid interval %q", iv.text)
	var err error
	if iv.fromRel {
		if out.from, err = er.fromEnd(iv.from, end, Lower); err != nil {
			return Interval[T, C]{}, invalid(errPrefix, err)
		}
		out.fromRel = false
	}
	if iv.toRel {
		if out.to, err = er.fromEnd(iv.to, end, Upper); err != nil {
			return Interval[T, C]{}, invalid(errPrefix, err)
		}
		out.toRel = false
	}
	if out.hasFrom && out.hasTo && c.Compare(out.to, out.from) <= 0 {
		return Interval[T, C]{}, invalid(errPrefix+": it is empty or ends before it starts", nil)
	}
	return out, nil
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

// Relative returns true if either end is relative to the end of the
// interval's items, see Resolve. From and To return the offset for such an
// end and Contains and Overlaps are false until the interval is resolved.
func (iv *Interval[T, C]) Relative() bool { return iv.unresolved() }

// From returns the start of the interval, and false if it is open.
func (iv *Interval[T, C]) From() (T, bool) { return iv.from, iv.hasFrom }

// To returns the end of the interval, which is not within it, and false if it
// is open.
func (iv *Interval[T, C]) To() (T, bool) { return iv.to, iv.hasTo }

// Contains returns true if v is within the interval. It is false if the
// interval is not set or is unresolved.
func (iv *Interval[T, C]) Contains(v T) bool {
	var c C
	if !iv.set || iv.unresolved() {
		return false
	}
	return (!iv.hasFrom || c.Compare(v, iv.from) >= 0) && (!iv.hasTo || c.Compare(v, iv.to) < 0)
}

// Overlaps returns true if the intervals have any value in common. It is
// false if either is nil, not set or unresolved.
func (iv *Interval[T, C]) Overlaps(other *Interval[T, C]) bool {
	var c C
	if iv == nil || other == nil || !iv.set || !other.set || iv.unresolved() || other.unresolved() {
		return false
	}
	before := func(a, b *Interval[T, C]) bool { // a lies entirely before b
		return a.hasTo && b.hasFrom && c.Compare(a.to, b.from) <= 0
	}
	return !before(iv, other) && !before(other, iv)
}

// MarshalText implements encoding.TextMarshaler.
func (iv Interval[T, C]) MarshalText() ([]byte, error) { return []byte(iv.text), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (iv *Interval[T, C]) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*iv = Interval[T, C]{sep: iv.sep}
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

func (TimeCodec) parseSpan(s string) (time.Time, time.Time, error) {
	return cmdtypes.DefaultTimeParser.Span(s)
}

func (TimeCodec) parseRange(lhs, rhs string) (time.Time, time.Time, error) {
	tp := cmdtypes.DefaultTimeParser
	now := tp.Now
	var fixed time.Time
	var snapped bool
	tp.Now = func() time.Time {
		if !snapped {
			if now != nil {
				fixed = now()
			} else {
				fixed = time.Now()
			}
			snapped = true
		}
		return fixed
	}
	from, _, err := tp.Span(lhs)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	_, to, err := tp.Span(rhs)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}

// Compare implements Codec.
func (TimeCodec) Compare(a, b time.Time) int { return a.Compare(b) }

func nextEdge[T ~int | ~int64](v, maxVal T, edge Edge) (T, error) {
	if edge == Upper {
		if v == maxVal {
			return 0, fmt.Errorf("value %v overflows upper bound", v)
		}
		return v + 1, nil
	}
	return v, nil
}

func spanFromPoint[T ~int | ~int64](v, maxVal T, err error) (T, T, error) {
	if err != nil {
		return 0, 0, err
	}
	upper, err := nextEdge(v, maxVal, Upper)
	if err != nil {
		return 0, 0, err
	}
	return v, upper, nil
}

// IntCodec is the Codec for int. It also supports values relative to the end
// of an Interval, see Interval.Resolve.
type IntCodec struct{}

func parseIntValue(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid int %q: %w", s, err)
	}
	return n, nil
}

// Parse implements Codec.
func (IntCodec) Parse(s string, edge Edge) (int, error) {
	n, err := parseIntValue(s)
	if err != nil {
		return 0, err
	}
	return nextEdge(n, math.MaxInt, edge)
}

func (IntCodec) parseOffset(s string) (int, error) {
	n, err := parseIntValue(s)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, fmt.Errorf("offset %d from the end must be at least 1", n)
	}
	return n, nil
}

func (IntCodec) fromEnd(offset, end int, edge Edge) (int, error) {
	v := end - offset
	if v < 0 {
		return 0, fmt.Errorf("^%d is before the first of %d items", offset, end)
	}
	return nextEdge(v, math.MaxInt, edge)
}

func (IntCodec) parseSpan(s string) (int, int, error) {
	n, err := parseIntValue(s)
	return spanFromPoint(n, math.MaxInt, err)
}

// Compare implements Codec.
func (IntCodec) Compare(a, b int) int { return cmp.Compare(a, b) }

// ByteSizeCodec is the Codec for cmdtypes.ByteSize, accepting the same
// notations as it.
type ByteSizeCodec struct{}

// Parse implements Codec.
func (ByteSizeCodec) Parse(s string, edge Edge) (cmdtypes.ByteSize, error) {
	n, err := cmdtypes.ParseByteSize(s)
	if err != nil {
		return 0, err
	}
	return nextEdge(n, cmdtypes.ByteSize(math.MaxInt64), edge)
}

func (ByteSizeCodec) parseSpan(s string) (cmdtypes.ByteSize, cmdtypes.ByteSize, error) {
	n, err := cmdtypes.ParseByteSize(s)
	return spanFromPoint(n, cmdtypes.ByteSize(math.MaxInt64), err)
}

// Compare implements Codec.
func (ByteSizeCodec) Compare(a, b cmdtypes.ByteSize) int { return cmp.Compare(a, b) }

// DurationCodec is the Codec for time.Duration, accepting the notations of
// cmdtypes.ParseDuration, with a leading '+' or '-' allowed.
type DurationCodec struct{}

func parseSignedDuration(s string) (time.Duration, error) {
	t := strings.TrimSpace(s)
	neg := false
	if rest, ok := strings.CutPrefix(t, "-"); ok {
		neg = true
		t = rest
	} else if rest, ok := strings.CutPrefix(t, "+"); ok {
		t = rest
	}
	d, err := cmdtypes.ParseDuration(t)
	if err != nil {
		return 0, err
	}
	if neg {
		d = -d
	}
	return d, nil
}

// Parse implements Codec.
func (DurationCodec) Parse(s string, edge Edge) (time.Duration, error) {
	d, err := parseSignedDuration(s)
	if err != nil {
		return 0, err
	}
	return nextEdge(d, time.Duration(math.MaxInt64), edge)
}

func (DurationCodec) parseSpan(s string) (time.Duration, time.Duration, error) {
	d, err := parseSignedDuration(s)
	return spanFromPoint(d, time.Duration(math.MaxInt64), err)
}

// Compare implements Codec.
func (DurationCodec) Compare(a, b time.Duration) int { return cmp.Compare(a, b) }

// StringCodec is the Codec for strings, which are ordered lexically. The
// span of a string s ends just after it, at s followed by a NUL.
type StringCodec struct{}

// Parse implements Codec.
func (StringCodec) Parse(s string, edge Edge) (string, error) {
	if s == "" {
		return "", errors.New("empty string")
	}
	if edge == Upper {
		return s + "\x00", nil
	}
	return s, nil
}

// Compare implements Codec.
func (StringCodec) Compare(a, b string) int { return strings.Compare(a, b) }

// ListSeparator separates the intervals of an Intervals.
const ListSeparator = ","

// Intervals is a flag.Value for a comma separated list of Interval, such as
// 1..3,7,^2.., that are all-or-nothing to set. The zero value is unset.
// Setting it replaces rather than adds to any previous value.
type Intervals[T any, C Codec[T]] struct {
	sep   string
	items []Interval[T, C]
}

// SetSeparator sets the separator used within each Interval, see
// Interval.SetSeparator. It must not be ListSeparator.
func (l *Intervals[T, C]) SetSeparator(sep string) { l.sep = sep }

// Set implements flag.Value.
func (l *Intervals[T, C]) Set(s string) error {
	var items []Interval[T, C]
	for part := range strings.SplitSeq(s, ListSeparator) {
		iv := Interval[T, C]{sep: l.sep}
		if err := iv.Set(strings.TrimSpace(part)); err != nil {
			return invalid(fmt.Sprintf("invalid list of intervals %q", s), err)
		}
		items = append(items, iv)
	}
	l.items = items
	return nil
}

// String implements flag.Value, returning the list as it was set.
func (l *Intervals[T, C]) String() string {
	if l == nil {
		return ""
	}
	parts := make([]string, len(l.items))
	for i := range l.items {
		parts[i] = l.items[i].String()
	}
	return strings.Join(parts, ListSeparator)
}

// Get implements flag.Getter, returning the list itself.
func (l *Intervals[T, C]) Get() any { return l }

// IsDefault returns true if the value has not been set.
func (l *Intervals[T, C]) IsDefault() bool { return len(l.items) == 0 }

// Items returns the intervals, which must not be modified.
func (l *Intervals[T, C]) Items() []Interval[T, C] { return l.items }

// Contains returns true if v is within any of the intervals.
func (l *Intervals[T, C]) Contains(v T) bool {
	for i := range l.items {
		if l.items[i].Contains(v) {
			return true
		}
	}
	return false
}

// Overlaps returns true if any of the intervals overlap other.
func (l *Intervals[T, C]) Overlaps(other *Interval[T, C]) bool {
	for i := range l.items {
		if l.items[i].Overlaps(other) {
			return true
		}
	}
	return false
}

// Resolve resolves each of the intervals, see Interval.Resolve.
func (l *Intervals[T, C]) Resolve(end T) (Intervals[T, C], error) {
	out := Intervals[T, C]{sep: l.sep, items: make([]Interval[T, C], len(l.items))}
	for i := range l.items {
		iv, err := l.items[i].Resolve(end)
		if err != nil {
			return Intervals[T, C]{}, err
		}
		out.items[i] = iv
	}
	return out, nil
}

// MarshalText implements encoding.TextMarshaler.
func (l Intervals[T, C]) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *Intervals[T, C]) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*l = Intervals[T, C]{sep: l.sep}
		return nil
	}
	return l.Set(string(text))
}

type (
	// StringBound is a Bound of a string.
	StringBound = Bound[string, StringCodec]
	// StringInterval is an Interval of strings.
	StringInterval = Interval[string, StringCodec]

	// TimeIntervals is an Intervals of times.
	TimeIntervals = Intervals[time.Time, TimeCodec]
	// IntIntervals is an Intervals of ints.
	IntIntervals = Intervals[int, IntCodec]
	// StringIntervals is an Intervals of strings.
	StringIntervals = Intervals[string, StringCodec]
	// ByteSizeIntervals is an Intervals of numbers of bytes.
	ByteSizeIntervals = Intervals[cmdtypes.ByteSize, ByteSizeCodec]
	// DurationIntervals is an Intervals of durations.
	DurationIntervals = Intervals[time.Duration, DurationCodec]

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
