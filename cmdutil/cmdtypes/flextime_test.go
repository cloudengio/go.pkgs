// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
)

func mustParse(t *testing.T, format, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(format, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

var flexTimeCases = []struct {
	in     string
	format string
}{
	{"2021-10-10", time.DateOnly},
	{"2021-10-10T03:03:03-07:00", time.RFC3339},
	{"03:03:05", time.TimeOnly},
	{"2021-10-10 03:03:05", time.DateTime},
}

func TestParseFlexTime(t *testing.T) {
	for _, tc := range flexTimeCases {
		got, err := cmdtypes.ParseFlexTime(tc.in)
		if err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if want := mustParse(t, tc.format, tc.in); !got.Time().Equal(want) {
			t.Errorf("%v: got %v, want %v", tc.in, got.Time(), want)
		}
	}
	if _, err := cmdtypes.ParseFlexTime("not-a-time"); err == nil {
		t.Error("expected an error for an unrecognised format")
	} else if !strings.Contains(err.Error(), "invalid time") {
		t.Errorf("got %v, want it to report an invalid time", err)
	}
}

// TestFlexTimeAllEncodings verifies that the one type decodes each accepted
// format from JSON and JSON v2, both of which route through UnmarshalText.
// The YAML equivalent is in cmdyaml, which is where this package's YAML
// behaviour is verified.
func TestFlexTimeAllEncodings(t *testing.T) {
	for _, tc := range flexTimeCases {
		want := mustParse(t, tc.format, tc.in)

		var fromJSON cmdtypes.FlexTime
		if err := json.Unmarshal([]byte(fmt.Sprintf("%q", tc.in)), &fromJSON); err != nil {
			t.Errorf("%v: json: %v", tc.in, err)
		} else if !fromJSON.Time().Equal(want) {
			t.Errorf("%v: json: got %v, want %v", tc.in, fromJSON.Time(), want)
		}

		var fromV2 cmdtypes.FlexTime
		if err := jsonv2.Unmarshal([]byte(fmt.Sprintf("%q", tc.in)), &fromV2); err != nil {
			t.Errorf("%v: json/v2: %v", tc.in, err)
		} else if !fromV2.Time().Equal(want) {
			t.Errorf("%v: json/v2: got %v, want %v", tc.in, fromV2.Time(), want)
		}
	}
}

// TestFlexTimeMarshalsAsRFC3339 verifies that whichever format a value was
// read from, it is written as RFC3339 and so can be read back.
func TestFlexTimeMarshalsAsRFC3339(t *testing.T) {
	for _, tc := range flexTimeCases {
		ft, err := cmdtypes.ParseFlexTime(tc.in)
		if err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		want := ft.Time().Format(time.RFC3339)
		if got := ft.String(); got != want {
			t.Errorf("%v: String: got %v, want %v", tc.in, got, want)
		}
		text, err := ft.MarshalText()
		if err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if got := string(text); got != want {
			t.Errorf("%v: MarshalText: got %v, want %v", tc.in, got, want)
		}
		for name, buf := range map[string][]byte{
			"json":    mustMarshal(t, func() ([]byte, error) { return json.Marshal(ft) }),
			"json/v2": mustMarshal(t, func() ([]byte, error) { return jsonv2.Marshal(ft) }),
		} {
			var back cmdtypes.FlexTime
			var err error
			switch name {
			case "json":
				err = json.Unmarshal(buf, &back)
			case "json/v2":
				err = jsonv2.Unmarshal(buf, &back)
			}
			if err != nil {
				t.Errorf("%v: %v: %v", tc.in, name, err)
				continue
			}
			if !back.Time().Equal(ft.Time()) {
				t.Errorf("%v: %v: got %v, want %v", tc.in, name, back.Time(), ft.Time())
			}
		}
	}
}

func mustMarshal(t *testing.T, fn func() ([]byte, error)) []byte {
	t.Helper()
	buf, err := fn()
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestFlexTimeErrors(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"json unrecognised", `"not-a-time"`},
		{"json non-string", `12345`},
		{"json empty", `""`},
	} {
		var ft cmdtypes.FlexTime
		if err := json.Unmarshal([]byte(tc.in), &ft); err == nil {
			t.Errorf("%v: expected an error, got %v", tc.name, ft.Time())
		}
	}
}

// TestFlexTimeNull records what each encoder does with a null, which differs:
// encoding/json leaves the value alone while json/v2 zeroes it.
func TestFlexTimeNull(t *testing.T) {
	base, err := cmdtypes.ParseFlexTime("2021-10-10T03:03:03Z")
	if err != nil {
		t.Fatal(err)
	}

	v1 := base
	if err := json.Unmarshal([]byte("null"), &v1); err != nil {
		t.Fatalf("json: %v", err)
	}
	if !v1.Time().Equal(base.Time()) {
		t.Errorf("json: got %v, want the value to be left alone", v1.Time())
	}

	v2 := base
	if err := jsonv2.Unmarshal([]byte("null"), &v2); err != nil {
		t.Fatalf("json/v2: %v", err)
	}
	if !v2.Time().IsZero() {
		t.Errorf("json/v2: got %v, want the zero time", v2.Time())
	}
}
