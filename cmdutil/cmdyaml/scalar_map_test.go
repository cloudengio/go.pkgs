// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
)

func TestVariablesLoad(t *testing.T) {
	spec := []byte(`
vars:
  host: localhost
  port: 8080
  ratio: 1.5
  enabled: true
  empty:
`)

	v := cmdyaml.NewVariables()
	if err := v.Load(spec, "vars"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct{ key, want string }{
		{"host", "localhost"},
		{"port", "8080"},
		{"ratio", "1.5"},
		{"enabled", "true"},
		{"empty", ""},
		{"missing", ""},
	}
	for _, tc := range cases {
		if got := v.Mapping(tc.key); got != tc.want {
			t.Errorf("Mapping(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestVariablesLoad_Accumulates(t *testing.T) {
	spec1 := []byte(`vars:
  a: one
  b: two
`)
	spec2 := []byte(`vars:
  b: overwritten
  c: three
`)

	v := cmdyaml.NewVariables()
	if err := v.Load(spec1, "vars"); err != nil {
		t.Fatalf("load spec1: %v", err)
	}
	if err := v.Load(spec2, "vars"); err != nil {
		t.Fatalf("load spec2: %v", err)
	}

	cases := []struct{ key, want string }{
		{"a", "one"},
		{"b", "overwritten"},
		{"c", "three"},
	}
	for _, tc := range cases {
		if got := v.Mapping(tc.key); got != tc.want {
			t.Errorf("ExpandEnv(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestVariablesLoad_AbsentMapName(t *testing.T) {
	spec := []byte(`other: value`)
	var v cmdyaml.Variables
	if err := v.Load(spec, "vars"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := v.Mapping("anything"); got != "" {
		t.Errorf("expected empty string for absent map, got %q", got)
	}
}

func TestVariablesLoad_RejectsNestedMap(t *testing.T) {
	spec := []byte(`
vars:
  host: localhost
  nested:
    key: value
`)
	var v cmdyaml.Variables
	if err := v.Load(spec, "vars"); err == nil {
		t.Fatal("expected error for nested map value, got nil")
	}
}

func TestVariablesLoad_RejectsSequence(t *testing.T) {
	spec := []byte(`
vars:
  host: localhost
  list:
    - a
    - b
`)
	var v cmdyaml.Variables
	if err := v.Load(spec, "vars"); err == nil {
		t.Fatal("expected error for sequence value, got nil")
	}
}

func TestVariablesLoad_RejectsNonMappingTarget(t *testing.T) {
	spec := []byte(`vars: not-a-map`)
	var v cmdyaml.Variables
	if err := v.Load(spec, "vars"); err == nil {
		t.Fatal("expected error when named key is not a mapping, got nil")
	}
}

func TestVariablesLoad_InvalidYAML(t *testing.T) {
	var v cmdyaml.Variables
	if err := v.Load([]byte(": : :"), "vars"); err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestVariablesExpandEnv_NilReceiver(t *testing.T) {
	var v *cmdyaml.Variables
	if got := v.Mapping("key"); got != "" {
		t.Errorf("nil receiver ExpandEnv: got %q, want %q", got, "")
	}
}
