// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
)

// colour implements EnumType for use in tests.
type colour int

const (
	Red colour = iota
	Green
	Blue
)

func (colour) EnumValues() map[string]colour {
	return map[string]colour{
		"red":   Red,
		"green": Green,
		"blue":  Blue,
	}
}

func ExampleEnum() {
	var e flags.Enum[colour]
	if err := e.Set("green"); err != nil {
		panic(err)
	}
	fmt.Println(e.String())
	fmt.Println(e.Value)
	// Output:
	// green
	// 1
}


func TestEnum(t *testing.T) {
	var e flags.Enum[colour]

	// Zero value: String returns the name for the zero value.
	if got, want := e.String(), "red"; got != want {
		t.Errorf("zero String: got %q, want %q", got, want)
	}

	// Set valid values.
	for _, tc := range []struct {
		input string
		want  colour
	}{
		{"red", Red},
		{"green", Green},
		{"blue", Blue},
	} {
		if err := e.Set(tc.input); err != nil {
			t.Errorf("Set(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if e.Value != tc.want {
			t.Errorf("Set(%q): got %v, want %v", tc.input, e.Value, tc.want)
		}
		if got := e.String(); got != tc.input {
			t.Errorf("String() after Set(%q): got %q", tc.input, got)
		}
	}

	// Set invalid value returns an error that names the invalid input and
	// lists the allowed values.
	err := e.Set("purple")
	if err == nil {
		t.Fatal("Set(invalid): expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "purple") {
		t.Errorf("error missing invalid value: %q", msg)
	}
	for _, valid := range []string{"red", "green", "blue"} {
		if !strings.Contains(msg, valid) {
			t.Errorf("error missing allowed value %q: %q", valid, msg)
		}
	}
	// Value must be unchanged after a failed Set.
	if e.Value != Blue {
		t.Errorf("value changed after failed Set: got %v, want Blue", e.Value)
	}

	// AllowedValues returns a sorted, comma-separated list.
	allowed := e.AllowedValues()
	if got, want := allowed, "blue, green, red"; got != want {
		t.Errorf("AllowedValues: got %q, want %q", got, want)
	}
}

// logLevel is a string-based comparable type showing that Enum works beyond
// integer underlying types.
type logLevel string

const (
	LevelDebug logLevel = "debug"
	LevelInfo  logLevel = "info"
	LevelError logLevel = "error"
)

func (logLevel) EnumValues() map[string]logLevel {
	return map[string]logLevel{
		"debug": LevelDebug,
		"info":  LevelInfo,
		"error": LevelError,
	}
}

func ExampleEnum_string() {
	var e flags.Enum[logLevel]
	if err := e.Set("info"); err != nil {
		panic(err)
	}
	fmt.Println(e.String())
	fmt.Println(e.Value)
	// Output:
	// info
	// info
}

func TestEnumString(t *testing.T) {
	var e flags.Enum[logLevel]

	// Zero value is logLevel(""), not in the map; String falls back to %v.
	if got, want := e.String(), ""; got != want {
		t.Errorf("zero String: got %q, want %q", got, want)
	}

	// Set valid values and check round-trip.
	for _, tc := range []struct {
		input string
		want  logLevel
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"error", LevelError},
	} {
		if err := e.Set(tc.input); err != nil {
			t.Errorf("Set(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if e.Value != tc.want {
			t.Errorf("Set(%q): got %v, want %v", tc.input, e.Value, tc.want)
		}
		if got := e.String(); got != tc.input {
			t.Errorf("String() after Set(%q): got %q", tc.input, got)
		}
	}

	// Invalid value: error must mention the bad input and leave Value unchanged.
	err := e.Set("trace")
	if err == nil {
		t.Fatal("Set(invalid): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "trace") {
		t.Errorf("error missing invalid value: %q", err.Error())
	}
	if e.Value != LevelError {
		t.Errorf("value changed after failed Set: got %v", e.Value)
	}

	// AllowedValues is sorted alphabetically.
	if got, want := e.AllowedValues(), "debug, error, info"; got != want {
		t.Errorf("AllowedValues: got %q, want %q", got, want)
	}
}
