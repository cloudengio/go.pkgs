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
	// A non-scalar entry must not prevent its valid siblings from being
	// registered.
	if got, want := v.Mapping("host"), "localhost"; got != want {
		t.Errorf("Mapping(%q) = %q, want %q", "host", got, want)
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
	if got, want := v.Mapping("host"), "localhost"; got != want {
		t.Errorf("Mapping(%q) = %q, want %q", "host", got, want)
	}
}

// TestVariablesLoad_AliasToScalar verifies that a vars entry whose value is
// an alias to a scalar anchor is resolved to that scalar's value, and that
// the presence of such an alias does not prevent other, plain entries in the
// same vars block from being registered (regression test: previously any
// non-ScalarNode value, including an AliasNode, caused the entire block to
// be dropped since the error was returned before any entries were
// committed).
func TestVariablesLoad_AliasToScalar(t *testing.T) {
	spec := []byte(`
base: &base hello
vars:
  x: *base
  y: world
`)
	v := cmdyaml.NewVariables()
	if err := v.Load(spec, "vars"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cases := []struct{ key, want string }{
		{"x", "hello"},
		{"y", "world"},
	}
	for _, tc := range cases {
		if got := v.Mapping(tc.key); got != tc.want {
			t.Errorf("Mapping(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// TestVariablesLoad_AliasToNonScalar verifies that an alias resolving to a
// non-scalar (e.g. a mapping) is still reported as an error, but, as with
// any other non-scalar entry, does not prevent its siblings from being
// registered.
func TestVariablesLoad_AliasToNonScalar(t *testing.T) {
	spec := []byte(`
base: &base
  nested: true
vars:
  x: *base
  y: world
`)
	v := cmdyaml.NewVariables()
	if err := v.Load(spec, "vars"); err == nil {
		t.Fatal("expected error for alias to non-scalar value, got nil")
	}
	if got, want := v.Mapping("y"), "world"; got != want {
		t.Errorf("Mapping(%q) = %q, want %q", "y", got, want)
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
