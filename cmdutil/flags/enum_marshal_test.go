// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
	"gopkg.in/yaml.v3"
)

// direction is a comparable type for YAML Enum tests.
type direction int

const (
	North direction = iota
	South
	East
	West
)

func (direction) EnumValues() map[string]direction {
	return map[string]direction{
		"north": North,
		"south": South,
		"east":  East,
		"west":  West,
	}
}

// priority is a comparable type for JSON Enum tests.
type priority int

const (
	Low priority = iota
	Medium
	High
)

func (priority) EnumValues() map[string]priority {
	return map[string]priority{
		"low":    Low,
		"medium": Medium,
		"high":   High,
	}
}

// speed is a string-based comparable type for JSON Enum tests.
type speed string

const (
	Slow   speed = "slow"
	Normal speed = "normal"
	Fast   speed = "fast"
)

func (speed) EnumValues() map[string]speed {
	return map[string]speed{
		"slow":   Slow,
		"normal": Normal,
		"fast":   Fast,
	}
}

// --- YAML ---

func Example_yamlEnumUnmarshal() {
	type Config struct {
		Dir flags.Enum[direction] `yaml:"direction"`
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte("direction: east\n"), &cfg); err != nil {
		panic(err)
	}
	fmt.Println(cfg.Dir.String())
	fmt.Println(cfg.Dir.Value)
	// Output:
	// east
	// 2
}

func Example_yamlEnumMarshal() {
	type Config struct {
		Dir flags.Enum[direction] `yaml:"direction"`
	}
	cfg := Config{}
	if err := cfg.Dir.Set("west"); err != nil {
		panic(err)
	}
	out, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))
	// Output:
	// direction: west
}

func TestCmdYAMLEnum(t *testing.T) {
	// Unmarshal valid values.
	for _, tc := range []struct {
		yaml string
		want direction
		name string
	}{
		{"north", North, "north"},
		{"south", South, "south"},
		{"east", East, "east"},
		{"west", West, "west"},
	} {
		var e flags.Enum[direction]
		if err := yaml.Unmarshal([]byte(tc.yaml+"\n"), &e); err != nil {
			t.Errorf("Unmarshal(%q): %v", tc.yaml, err)
			continue
		}
		if e.Value != tc.want {
			t.Errorf("Unmarshal(%q): got %v, want %v", tc.yaml, e.Value, tc.want)
		}
		if got := e.String(); got != tc.name {
			t.Errorf("String() after Unmarshal(%q): got %q, want %q", tc.yaml, got, tc.name)
		}
	}

	// Unmarshal invalid value returns an error naming the bad input.
	var e flags.Enum[direction]
	err := yaml.Unmarshal([]byte("diagonal\n"), &e)
	if err == nil {
		t.Fatal("Unmarshal(invalid): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "diagonal") {
		t.Errorf("error missing invalid value: %q", err.Error())
	}

	// Marshal round-trip: Set then MarshalYAML produces the string name.
	if err := e.Set("south"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	out, err := yaml.Marshal(&e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := strings.TrimSpace(string(out)), "south"; got != want {
		t.Errorf("Marshal: got %q, want %q", got, want)
	}

	// Full struct round-trip.
	type Config struct {
		Dir flags.Enum[direction] `yaml:"direction"`
	}
	cfg := Config{}
	if err := cfg.Dir.Set("west"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("Marshal struct: %v", err)
	}
	var cfg2 Config
	if err := yaml.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("Unmarshal struct: %v", err)
	}
	if cfg2.Dir.Value != West {
		t.Errorf("round-trip: got %v, want West", cfg2.Dir.Value)
	}
}

func TestCmdYAMLEnumEmbedsFlagsEnum(t *testing.T) {
	var e flags.Enum[direction]
	if err := e.Set("north"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got, want := e.String(), "north"; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
}

// --- JSON ---

func Example_jsonEnumMarshal() {
	type Config struct {
		P flags.Enum[priority] `json:"priority"`
	}
	cfg := Config{}
	if err := cfg.P.Set("high"); err != nil {
		panic(err)
	}
	out, err := json.Marshal(&cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	// Output:
	// {"priority":"high"}
}

func Example_jsonEnumUnmarshal() {
	type Config struct {
		P flags.Enum[priority] `json:"priority"`
	}
	var cfg Config
	if err := json.Unmarshal([]byte(`{"priority":"medium"}`), &cfg); err != nil {
		panic(err)
	}
	fmt.Println(cfg.P.String())
	fmt.Println(cfg.P.Value)
	// Output:
	// medium
	// 1
}

func TestCmdJSONEnum(t *testing.T) {
	// Unmarshal valid values.
	for _, tc := range []struct {
		json string
		want priority
		name string
	}{
		{`"low"`, Low, "low"},
		{`"medium"`, Medium, "medium"},
		{`"high"`, High, "high"},
	} {
		var e flags.Enum[priority]
		if err := json.Unmarshal([]byte(tc.json), &e); err != nil {
			t.Errorf("Unmarshal(%q): %v", tc.json, err)
			continue
		}
		if e.Value != tc.want {
			t.Errorf("Unmarshal(%q): got %v, want %v", tc.json, e.Value, tc.want)
		}
		if got := e.String(); got != tc.name {
			t.Errorf("String() after Unmarshal(%q): got %q, want %q", tc.json, got, tc.name)
		}
	}

	// Unmarshal invalid value returns an error naming the bad input.
	var e flags.Enum[priority]
	err := json.Unmarshal([]byte(`"critical"`), &e)
	if err == nil {
		t.Fatal("Unmarshal(invalid): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "critical") {
		t.Errorf("error missing invalid value: %q", err.Error())
	}

	// Unmarshal non-string JSON returns an error.
	err = json.Unmarshal([]byte(`42`), &e)
	if err == nil {
		t.Fatal("Unmarshal(non-string): expected error, got nil")
	}

	// Marshal round-trip via MarshalJSON.
	if err := e.Set("high"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	out, err := json.Marshal(&e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(out), `"high"`; got != want {
		t.Errorf("Marshal: got %q, want %q", got, want)
	}

	// Full struct round-trip.
	type Config struct {
		P flags.Enum[priority] `json:"priority"`
	}
	cfg := Config{}
	if err := cfg.P.Set("medium"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("Marshal struct: %v", err)
	}
	var cfg2 Config
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("Unmarshal struct: %v", err)
	}
	if cfg2.P.Value != Medium {
		t.Errorf("round-trip: got %v, want Medium", cfg2.P.Value)
	}
}

func TestCmdJSONEnumString(t *testing.T) {
	// Non-integer comparable type (speed string).
	var e flags.Enum[speed]

	// Zero value is speed(""), not in map; String falls back to %v.
	if got, want := e.String(), ""; got != want {
		t.Errorf("zero String: got %q, want %q", got, want)
	}

	// Unmarshal valid string-based values.
	for _, tc := range []struct {
		json string
		want speed
	}{
		{`"slow"`, Slow},
		{`"normal"`, Normal},
		{`"fast"`, Fast},
	} {
		if err := json.Unmarshal([]byte(tc.json), &e); err != nil {
			t.Errorf("Unmarshal(%q): %v", tc.json, err)
			continue
		}
		if e.Value != tc.want {
			t.Errorf("Unmarshal(%q): got %v, want %v", tc.json, e.Value, tc.want)
		}
	}

	// Marshal round-trip.
	if err := e.Set("slow"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	out, err := json.Marshal(&e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(out), `"slow"`; got != want {
		t.Errorf("Marshal: got %q, want %q", got, want)
	}
}

func TestCmdJSONEnumEmbedsFlagsEnum(t *testing.T) {
	var e flags.Enum[priority]
	if err := e.Set("low"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got, want := e.String(), "low"; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
}
