// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes

import (
	"fmt"
	"math"
	"math/bits"
	"regexp"
	"strings"
	"time"
)

// TimeParser parses the times and dates that a person writes on a command
// line. Unlike FlexTime, which is read from and written to configuration
// files and so is exact and unambiguous, it favours convenience: a value may
// be coarse, "2024-03" for a month, or relative, "-7d" for a week ago, and a
// value that gives no time zone is in Location rather than UTC.
//
// The zero value uses time.Local and time.Now.
type TimeParser struct {
	// Location is the time zone of a value that does not give one, and the
	// zone that days, months and years begin in. It defaults to time.Local.
	Location *time.Location
	// Now returns the current time, which relative values are relative to.
	// It defaults to time.Now.
	Now func() time.Time
}

// DefaultTimeParser is used by the flags that parse times. Tests may replace
// it sequentially during setup to make relative values and zones repeatable;
// it must not be modified concurrently with flag parsing.
var DefaultTimeParser TimeParser

func (p TimeParser) location() *time.Location {
	if p.Location == nil {
		return time.Local
	}
	return p.Location
}

func (p TimeParser) now() time.Time {
	if p.Now == nil {
		return time.Now().In(p.location())
	}
	return p.Now().In(p.location())
}

// unit is the precision of a value, and so the length of the span it names.
type unit int

const (
	second unit = iota
	minute
	day
	month
	year
)

func (u unit) next(t time.Time) time.Time {
	switch u {
	case second:
		return t.Add(time.Second)
	case minute:
		return t.Add(time.Minute)
	case day:
		return t.AddDate(0, 0, 1)
	case month:
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(1, 0, 0)
}

// layouts are tried in order, those with a zone first. A layout without a
// zone is read in the parser's Location.
var layouts = []struct {
	layout string
	unit   unit
}{
	{time.RFC3339, second},
	{"2006-01-02T15:04:05", second},
	{time.DateTime, second},
	{"2006-01-02T15:04", minute},
	{"2006-01-02 15:04", minute},
	{time.DateOnly, day},
	{"2006-01", month},
	{"2006", year},
}

// timeOnlyLayouts name a time of day, which is taken to be today's.
var timeOnlyLayouts = []struct {
	layout string
	unit   unit
}{
	{time.TimeOnly, second},
	{"15:04", minute},
}

// TimeSpanFormats describes what Span accepts, for use in usage messages.
const TimeSpanFormats = `a date and time such as 2024-03-05T10:30:00Z, "2024-03-05 10:30" or 10:30 (today), ` +
	`a date, month or year such as 2024-03-05, 2024-03 or 2024, ` +
	`or a time relative to now such as now, today, yesterday, -7d, now-1h or "2w ago" (units: ns us ms s m h d w)`

// Span returns the span of time that s names, [start, end): a value names all
// of the time to its precision, so that the date 2024-03-05 is the whole of
// that day and the month 2024-03 the whole of that month, and a value to the
// second is that second. start is the lower edge and end the upper, exclusive,
// edge; a range of times uses the first of its beginning and the second of its end,
// so that "2024-03-01..2024-03-05" includes all of the fifth.
//
// A relative value is relative to the current time and is truncated to the
// second, except expressions anchored at today, yesterday or tomorrow, which
// preserve day-length spans (so "today+9h" spans 24 hours starting at 09:00).
// A day is 24 hours and a week seven of them, whatever the clock does.
func (p TimeParser) Span(s string) (start, end time.Time, err error) {
	s = strings.TrimSpace(s)
	loc := p.location()
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l.layout, s, loc); err == nil {
			return t, l.unit.next(t), nil
		}
	}
	for _, l := range timeOnlyLayouts {
		if t, err := time.ParseInLocation(l.layout, s, loc); err == nil {
			now := p.now()
			t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
			return t, l.unit.next(t), nil
		}
	}
	if start, u, ok, err := p.relative(s); err != nil {
		return time.Time{}, time.Time{}, err
	} else if ok {
		return start, u.next(start), nil
	}
	return time.Time{}, time.Time{}, fmt.Errorf("invalid time %q: use %s", s, TimeSpanFormats)
}

// Parse returns the start of the span that s names, see Span.
func (p TimeParser) Parse(s string) (time.Time, error) {
	start, _, err := p.Span(s)
	return start, err
}

var agoRE = regexp.MustCompile(`^(\S+)\s+ago$`)

var durationUnits = map[string]time.Duration{
	"ns": time.Nanosecond, "us": time.Microsecond, "µs": time.Microsecond, "μs": time.Microsecond,
	"ms": time.Millisecond, "s": time.Second, "m": time.Minute, "h": time.Hour,
	"d": 24 * time.Hour, "w": 7 * 24 * time.Hour,
}

func scanDigits(s string, i int, v, scale uint64, isFrac bool) (next int, nv, nscale uint64, count int) {
	const maxVal = uint64(math.MaxInt64)
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		count++
		d := uint64(s[i] - '0')
		if isFrac {
			if scale <= maxVal/10 && v <= maxVal/10 {
				v = v*10 + d
				scale *= 10
			}
			continue
		}
		if v > (maxVal-d)/10 {
			return i, 0, 0, -1
		}
		v = v*10 + d
	}
	return i, v, scale, count
}

func parseDurationComponent(s string, i int) (part uint64, next int, err error) {
	const maxVal = uint64(math.MaxInt64)
	i, intPart, _, nInt := scanDigits(s, i, 0, 1, false)
	if nInt < 0 {
		return 0, 0, fmt.Errorf("invalid duration %q: overflow", s)
	}
	if nInt == 0 {
		return 0, 0, fmt.Errorf("invalid duration %q: use numbers with the units ns us ms s m h d w, such as 1h30m or 2w", s)
	}
	var fracPart uint64
	scale := uint64(1)
	if i < len(s) && s[i] == '.' {
		i++
		var nFrac int
		i, fracPart, scale, nFrac = scanDigits(s, i, 0, 1, true)
		if nFrac <= 0 {
			return 0, 0, fmt.Errorf("invalid duration %q: use numbers with the units ns us ms s m h d w, such as 1h30m or 2w", s)
		}
	}
	uStart := i
	for i < len(s) && (s[i] < '0' || s[i] > '9') {
		i++
	}
	unitDur, ok := durationUnits[s[uStart:i]]
	if !ok {
		return 0, 0, fmt.Errorf("invalid duration %q: use numbers with the units ns us ms s m h d w, such as 1h30m or 2w", s)
	}
	u := uint64(unitDur)
	if intPart > maxVal/u {
		return 0, 0, fmt.Errorf("invalid duration %q: overflow", s)
	}
	part = intPart * u
	if fracPart > 0 {
		hi, lo := bits.Mul64(fracPart, u)
		q, _ := bits.Div64(hi, lo, scale)
		if q > maxVal-part {
			return 0, 0, fmt.Errorf("invalid duration %q: overflow", s)
		}
		part += q
	}
	return part, i, nil
}

// ParseDuration is like time.ParseDuration, with the addition of the units d
// (24 hours) and w (7 days). It does not accept a sign.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("invalid duration %q: use numbers with the units ns us ms s m h d w, such as 1h30m or 2w", s)
	}
	const maxVal = uint64(math.MaxInt64)
	var total uint64
	for i := 0; i < len(s); {
		part, next, err := parseDurationComponent(s, i)
		if err != nil {
			return 0, err
		}
		if part > maxVal-total {
			return 0, fmt.Errorf("invalid duration %q: overflow", s)
		}
		total += part
		i = next
	}
	return time.Duration(total), nil
}

// relative parses a time relative to now, ok is false if s is not of that form.
func (p TimeParser) relative(s string) (start time.Time, u unit, ok bool, err error) {
	now := p.now()
	loc := p.location()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	base, u := now.Truncate(time.Second), second
	rest := s
	switch {
	case strings.HasPrefix(s, "now"):
		rest = s[len("now"):]
	case strings.HasPrefix(s, "today"):
		base, u, rest = midnight, day, s[len("today"):]
	case strings.HasPrefix(s, "yesterday"):
		base, u, rest = midnight.AddDate(0, 0, -1), day, s[len("yesterday"):]
	case strings.HasPrefix(s, "tomorrow"):
		base, u, rest = midnight.AddDate(0, 0, 1), day, s[len("tomorrow"):]
	case strings.HasPrefix(s, "-"), strings.HasPrefix(s, "+"):
	default:
		m := agoRE.FindStringSubmatch(s)
		if m == nil {
			return time.Time{}, 0, false, nil
		}
		rest = "-" + m[1]
	}
	if rest == "" {
		return base, u, true, nil
	}
	if rest[0] != '-' && rest[0] != '+' {
		// Something that merely starts with a keyword, not an expression.
		return time.Time{}, 0, false, nil
	}
	d, err := ParseDuration(rest[1:])
	if err != nil {
		return time.Time{}, 0, false, fmt.Errorf("invalid time %q: %w", s, err)
	}
	if rest[0] == '-' {
		d = -d
	}
	return base.Add(d), u, true, nil
}
