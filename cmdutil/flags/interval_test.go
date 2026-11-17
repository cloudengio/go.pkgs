// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"encoding/json"
	"flag"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
	"cloudeng.io/cmdutil/flags"
	"gopkg.in/yaml.v3"
)

func useTestParser(t *testing.T) *time.Location {
	t.Helper()
	loc := time.FixedZone("test", -7*3600)
	orig := cmdtypes.DefaultTimeParser
	cmdtypes.DefaultTimeParser = cmdtypes.TimeParser{
		Location: loc,
		Now:      func() time.Time { return time.Date(2024, 3, 5, 10, 30, 15, 0, loc) },
	}
	t.Cleanup(func() { cmdtypes.DefaultTimeParser = orig })
	return loc
}

func TestTimeInterval(t *testing.T) {
	loc := useTestParser(t)
	d := func(m time.Month, day, h int) time.Time { return time.Date(2024, m, day, h, 0, 0, 0, loc) }
	for _, tc := range []struct {
		in             string
		from, to       time.Time
		hasFrom, hasTo bool
		inside         []time.Time
		outside        []time.Time
	}{
		// The end of a range includes all of the day that it names.
		{"2024-03-01..2024-03-05", d(3, 1, 0), d(3, 6, 0), true, true,
			[]time.Time{d(3, 1, 0), d(3, 5, 23)}, []time.Time{d(2, 29, 23), d(3, 6, 0)}},
		{"2024-03-05..", d(3, 5, 0), time.Time{}, true, false,
			[]time.Time{d(3, 5, 0), d(12, 1, 0)}, []time.Time{d(3, 4, 23)}},
		{"..2024-03-05", time.Time{}, d(3, 6, 0), false, true,
			[]time.Time{d(1, 1, 0), d(3, 5, 23)}, []time.Time{d(3, 6, 0)}},
		// A single value is all of it, at its own precision.
		{"2024-03-05", d(3, 5, 0), d(3, 6, 0), true, true,
			[]time.Time{d(3, 5, 0), d(3, 5, 23)}, []time.Time{d(3, 4, 23), d(3, 6, 0)}},
		{"2024-03", d(3, 1, 0), d(4, 1, 0), true, true,
			[]time.Time{d(3, 31, 23)}, []time.Time{d(4, 1, 0)}},
		// Relative values, with a separator that they do not contain.
		{"-7d..now", d(2, 27, 0).Add(10*time.Hour + 30*time.Minute + 15*time.Second), d(3, 5, 10).Add(30*time.Minute + 16*time.Second), true, true, nil, nil},
		{"yesterday..today", d(3, 4, 0), d(3, 6, 0), true, true,
			[]time.Time{d(3, 4, 0), d(3, 5, 12)}, []time.Time{d(3, 6, 0)}},
	} {
		var iv flags.TimeInterval
		if err := iv.Set(tc.in); err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if iv.String() != tc.in || iv.IsDefault() {
			t.Errorf("%q: String=%q, IsDefault=%v", tc.in, iv.String(), iv.IsDefault())
		}
		from, hasFrom := iv.From()
		to, hasTo := iv.To()
		if hasFrom != tc.hasFrom || hasTo != tc.hasTo || !from.Equal(tc.from) || !to.Equal(tc.to) {
			t.Errorf("%q: got [%v (%v), %v (%v)), want [%v, %v)", tc.in, from, hasFrom, to, hasTo, tc.from, tc.to)
		}
		for _, v := range tc.inside {
			if !iv.Contains(v) {
				t.Errorf("%q: %v should be inside", tc.in, v)
			}
		}
		for _, v := range tc.outside {
			if iv.Contains(v) {
				t.Errorf("%q: %v should be outside", tc.in, v)
			}
		}
	}
}

func TestIntervalErrors(t *testing.T) {
	useTestParser(t)
	for _, in := range []string{
		"", "..", "2024-03-05..2024-03-01", "2024-03-05..2024-03-05T00:00:00..", "x..2024", "2024..y",
	} {
		var iv flags.TimeInterval
		if err := iv.Set(in); err == nil {
			t.Errorf("%q: expected an error", in)
		} else if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("%q: unhelpful error: %v", in, err)
		}
		if !iv.IsDefault() || iv.Contains(time.Now()) {
			t.Errorf("%q: a failed Set changed the value", in)
		}
	}
	// A failed Set leaves the previous value in place.
	var iv flags.IntInterval
	if err := iv.Set("1..5"); err != nil {
		t.Fatal(err)
	}
	if err := iv.Set("9..2"); err == nil {
		t.Error("expected an error")
	}
	if iv.String() != "1..5" {
		t.Errorf("got %q", iv.String())
	}
}

func TestIntInterval(t *testing.T) {
	for _, tc := range []struct {
		in      string
		in1     []int
		outside []int
	}{
		{"3..5", []int{3, 4, 5}, []int{2, 6}}, // the end is included
		{"-5..-3", []int{-5, -4, -3}, []int{-6, -2}},
		{"3..", []int{3, 1000}, []int{2}},
		{"..3", []int{-1000, 3}, []int{4}},
		{"7", []int{7}, []int{6, 8}},
		{"+7", []int{7}, []int{6, 8}},
		{"5..5", []int{5}, []int{4, 6}},
	} {
		var iv flags.IntInterval
		if err := iv.Set(tc.in); err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		for _, v := range tc.in1 {
			if !iv.Contains(v) {
				t.Errorf("%q: %v should be inside", tc.in, v)
			}
		}
		for _, v := range tc.outside {
			if iv.Contains(v) {
				t.Errorf("%q: %v should be outside", tc.in, v)
			}
		}
	}
	for _, in := range []string{"a", "1.5", "1...3", "1..a", "5..3"} {
		var iv flags.IntInterval
		if err := iv.Set(in); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}

func TestSizeAndDurationIntervals(t *testing.T) {
	var sz flags.ByteSizeInterval
	if err := sz.Set("1KiB..1MiB"); err != nil {
		t.Fatal(err)
	}
	if !sz.Contains(cmdtypes.ByteSize(1024)) || !sz.Contains(cmdtypes.ByteSize(1<<20)) || sz.Contains(cmdtypes.ByteSize(1<<20+1)) || sz.Contains(1023) {
		t.Error("unexpected size membership")
	}
	var du flags.DurationInterval
	if err := du.Set("1h..2d"); err != nil {
		t.Fatal(err)
	}
	if !du.Contains(90*time.Minute) || !du.Contains(48*time.Hour) || du.Contains(49*time.Hour) {
		t.Error("unexpected duration membership")
	}
}

func TestBound(t *testing.T) {
	loc := useTestParser(t)
	var since flags.TimeBound
	if !since.IsDefault() || since.Contains(time.Now()) || since.Before(time.Now()) || since.After(time.Now()) {
		t.Error("an unset bound must not match anything")
	}
	if err := since.Set("2024-03-05"); err != nil {
		t.Fatal(err)
	}
	at := func(day, h int) time.Time { return time.Date(2024, 3, day, h, 0, 0, 0, loc) }
	for _, tc := range []struct {
		v                       time.Time
		contains, before, after bool
	}{
		{at(4, 23), false, false, true},
		{at(5, 0), true, false, false},
		{at(5, 23), true, false, false},
		{at(6, 0), false, true, false},
	} {
		if since.Contains(tc.v) != tc.contains || since.Before(tc.v) != tc.before || since.After(tc.v) != tc.after {
			t.Errorf("%v: got contains=%v before=%v after=%v", tc.v, since.Contains(tc.v), since.Before(tc.v), since.After(tc.v))
		}
	}
	if !since.Lower().Equal(at(5, 0)) || !since.Upper().Equal(at(6, 0)) {
		t.Errorf("got %v %v", since.Lower(), since.Upper())
	}
	var size flags.ByteSizeBound
	if err := size.Set("1MiB"); err != nil || !size.After(cmdtypes.ByteSize(1<<20-1)) || !size.Before(cmdtypes.ByteSize(1<<20+1)) {
		t.Errorf("unexpected size bound: %v", err)
	}
}

func TestOverlaps(t *testing.T) {
	parse := func(s string) *flags.IntInterval {
		iv := &flags.IntInterval{}
		if err := iv.Set(s); err != nil {
			t.Fatal(err)
		}
		return iv
	}
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"1..5", "5..9", true}, {"1..4", "5..9", false}, {"5..9", "1..4", false},
		{"..3", "3..", true}, {"..3", "4..", false}, {"1..", "..0", false}, {"1..", "5..", true},
	} {
		if got := parse(tc.a).Overlaps(parse(tc.b)); got != tc.want {
			t.Errorf("%v overlaps %v: got %v", tc.a, tc.b, got)
		}
	}
	if (&flags.IntInterval{}).Overlaps(parse("1..2")) || parse("1..2").Overlaps(&flags.IntInterval{}) {
		t.Error("an unset interval overlaps nothing")
	}
}

func TestIntervalFlags(t *testing.T) {
	loc := useTestParser(t)
	var args struct {
		Modified flags.TimeInterval  `cmdline:"modified,2024-03,files modified in this interval"`
		Since    flags.TimeBound     `cmdline:"since,,only files since this time"`
		Size     flags.ByteSizeBound `cmdline:"size,,size"`
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := flags.RegisterFlagsInStruct(fs, "cmdline", &args, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := args.Modified.String(), "2024-03"; got != want {
		t.Errorf("default: got %q, want %q", got, want)
	}
	if !args.Since.IsDefault() {
		t.Error("expected unset")
	}
	if err := fs.Parse([]string{"-modified", "-7d..now", "-since=yesterday", "-size", "2GB"}); err != nil {
		t.Fatal(err)
	}
	from, _ := args.Modified.From()
	if want := time.Date(2024, 2, 27, 10, 30, 15, 0, loc); !from.Equal(want) {
		t.Errorf("got %v, want %v", from, want)
	}
	if !args.Since.Lower().Equal(time.Date(2024, 3, 4, 0, 0, 0, 0, loc)) {
		t.Errorf("got %v", args.Since.Lower())
	}
	if err := fs.Parse([]string{"-modified", "bad"}); err == nil {
		t.Error("expected an error")
	}
}

func TestIntervalText(t *testing.T) {
	useTestParser(t)
	type cfg struct {
		Window flags.TimeInterval `yaml:"window" json:"window"`
		Max    flags.IntBound     `yaml:"max" json:"max"`
	}
	var c cfg
	if err := yaml.Unmarshal([]byte("window: -7d..now\nmax: 12\n"), &c); err != nil {
		t.Fatal(err)
	}
	if c.Window.String() != "-7d..now" || c.Max.Lower() != 12 || c.Max.Upper() != 13 {
		t.Errorf("unexpected: %+v", c)
	}
	// The text, relative values included, is what is written, and read back.
	out, err := yaml.Marshal(c)
	if err != nil || !strings.Contains(string(out), "window: -7d..now") {
		t.Fatalf("%s %v", out, err)
	}
	js, err := json.Marshal(c)
	if err != nil || !strings.Contains(string(js), `"window":"-7d..now"`) {
		t.Fatalf("%s %v", js, err)
	}
	var back cfg
	if err := json.Unmarshal(js, &back); err != nil || back.Window.String() != "-7d..now" || back.Window.IsDefault() {
		t.Errorf("%+v %v", back, err)
	}
	if err := yaml.Unmarshal([]byte("window: nonsense"), &back); err == nil {
		t.Error("expected an error")
	}
	var empty cfg
	if err := yaml.Unmarshal([]byte("window: ''"), &empty); err != nil || !empty.Window.IsDefault() {
		t.Errorf("%+v %v", empty, err)
	}
}
