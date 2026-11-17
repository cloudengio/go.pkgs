// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
)

func testParser() cmdtypes.TimeParser {
	loc := time.FixedZone("test", -7*3600)
	return cmdtypes.TimeParser{
		Location: loc,
		Now:      func() time.Time { return time.Date(2024, 3, 5, 10, 30, 15, 500, loc) },
	}
}

func TestTimeSpan(t *testing.T) {
	p := testParser()
	loc := p.Location
	at := func(y int, m time.Month, d, h, mi, s int) time.Time { return time.Date(y, m, d, h, mi, s, 0, loc) }
	for _, tc := range []struct {
		in         string
		start, end time.Time
	}{
		{"2024-03-05T10:30:00Z", time.Date(2024, 3, 5, 10, 30, 0, 0, time.UTC), time.Date(2024, 3, 5, 10, 30, 1, 0, time.UTC)},
		{"2024-03-05T10:30:00", at(2024, 3, 5, 10, 30, 0), at(2024, 3, 5, 10, 30, 1)},
		{"2024-03-05 10:30:00", at(2024, 3, 5, 10, 30, 0), at(2024, 3, 5, 10, 30, 1)},
		{"2024-03-05 10:30", at(2024, 3, 5, 10, 30, 0), at(2024, 3, 5, 10, 31, 0)},
		{"2024-03-05T10:30", at(2024, 3, 5, 10, 30, 0), at(2024, 3, 5, 10, 31, 0)},
		{"2024-03-05", at(2024, 3, 5, 0, 0, 0), at(2024, 3, 6, 0, 0, 0)},
		{"2024-03", at(2024, 3, 1, 0, 0, 0), at(2024, 4, 1, 0, 0, 0)},
		{"2024-12", at(2024, 12, 1, 0, 0, 0), at(2025, 1, 1, 0, 0, 0)},
		{"2024", at(2024, 1, 1, 0, 0, 0), at(2025, 1, 1, 0, 0, 0)},
		{"09:15", at(2024, 3, 5, 9, 15, 0), at(2024, 3, 5, 9, 16, 0)},
		{"09:15:20", at(2024, 3, 5, 9, 15, 20), at(2024, 3, 5, 9, 15, 21)},
		{"now", at(2024, 3, 5, 10, 30, 15), at(2024, 3, 5, 10, 30, 16)},
		{"today", at(2024, 3, 5, 0, 0, 0), at(2024, 3, 6, 0, 0, 0)},
		{"yesterday", at(2024, 3, 4, 0, 0, 0), at(2024, 3, 5, 0, 0, 0)},
		{"tomorrow", at(2024, 3, 6, 0, 0, 0), at(2024, 3, 7, 0, 0, 0)},
		{"-1h", at(2024, 3, 5, 9, 30, 15), at(2024, 3, 5, 9, 30, 16)},
		{"now-1h30m", at(2024, 3, 5, 9, 0, 15), at(2024, 3, 5, 9, 0, 16)},
		{"+2d", at(2024, 3, 7, 10, 30, 15), at(2024, 3, 7, 10, 30, 16)},
		{"2w ago", at(2024, 2, 20, 10, 30, 15), at(2024, 2, 20, 10, 30, 16)},
		{"today-7d", at(2024, 2, 27, 0, 0, 0), at(2024, 2, 28, 0, 0, 0)},
		{" 2024-03-05 ", at(2024, 3, 5, 0, 0, 0), at(2024, 3, 6, 0, 0, 0)},
	} {
		start, end, err := p.Span(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if !start.Equal(tc.start) || !end.Equal(tc.end) {
			t.Errorf("%q: got [%v, %v), want [%v, %v)", tc.in, start, end, tc.start, tc.end)
		}
	}
}

func TestTimeSpanErrors(t *testing.T) {
	p := testParser()
	for _, in := range []string{"", "tomorrowish", "nowhere", "-", "-7x", "7d ago ago", "2024-13", "2024-02-30", "yesterday7d", "ago", "-1.5.5h"} {
		if _, _, err := p.Span(in); err == nil {
			t.Errorf("%q: expected an error", in)
		} else if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("%q: unhelpful error: %v", in, err)
		}
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"90s": 90 * time.Second, "1h30m": 90 * time.Minute, "2d": 48 * time.Hour,
		"1w": 7 * 24 * time.Hour, "1w2d3h": (9*24 + 3) * time.Hour, "1.5h": 90 * time.Minute, "10ms": 10 * time.Millisecond,
		"100µs":                 100 * time.Microsecond,
		"100μs":                 100 * time.Microsecond,
		"105d1ns":               105*24*time.Hour + time.Nanosecond,
		"9223372036854775807ns": time.Duration(math.MaxInt64),
	} {
		if got, err := cmdtypes.ParseDuration(in); err != nil || got != want {
			t.Errorf("%q: got %v, %v, want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "5", "h", "-1h", "1h 30m", "1y", ".5h", "1.", "1.h", "1h-2m", "20000w", "9223372036854775808ns", "99999999999999999999ns"} {
		if _, err := cmdtypes.ParseDuration(in); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}

func TestDefaultTimeParserZero(t *testing.T) {
	// The zero parser works: local time and the real clock.
	got, err := cmdtypes.TimeParser{}.Parse("2024-03-05")
	if err != nil || !got.Equal(time.Date(2024, 3, 5, 0, 0, 0, 0, time.Local)) {
		t.Errorf("got %v, %v", got, err)
	}
	if _, err := (cmdtypes.TimeParser{}).Parse("now"); err != nil {
		t.Error(err)
	}
}
