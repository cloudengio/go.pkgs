// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile_test

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/envfile"
)

// Mapping is usable wherever a mapping function is called for, such as
// os.Expand or cmdyaml.WithExpandMapping.
var _ func(string) string = (&envfile.Variables{}).Mapping

const sampleEnvfile = `
# deployment settings
export REGION=us-west-2
BUCKET='app-logs'
EMPTY=
TMPL="$BUCKET/$REGION"
`

func writeEnvfile(t *testing.T, contents string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(name, []byte(contents), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func TestRead(t *testing.T) {
	vars, err := envfile.Read(strings.NewReader(sampleEnvfile))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := env("REGION", "us-west-2", "BUCKET", "app-logs", "EMPTY", "", "TMPL", "$BUCKET/$REGION")
	if got := vars.All(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := vars.Len(), 4; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := vars.Mapping("REGION"), "us-west-2"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// All returns a copy, so mutating it does not affect vars.
	all := vars.All()
	all["REGION"] = "elsewhere"
	delete(all, "BUCKET")
	if got, want := vars.Mapping("REGION"), "us-west-2"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := vars.Mapping("BUCKET"), "app-logs"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadFile(t *testing.T) {
	vars, err := envfile.ReadFile(writeEnvfile(t, sampleEnvfile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got, want := vars.Mapping("BUCKET"), "app-logs"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if _, err := envfile.ReadFile(filepath.Join(t.TempDir(), "no-such-file")); err == nil {
		t.Error("expected an error for a file that does not exist")
	}
}

func TestReadErrors(t *testing.T) {
	// Parse errors are returned as is, and no Variables is returned with them.
	for _, tc := range []struct {
		contents, errMsg string
	}{
		{"NOT A NAME=value\n", `invalid variable name`},
		{"NAME='unterminated\n", `unterminated single-quoted value`},
		{"NAME=\"unterminated\n", `unterminated double-quoted value`},
	} {
		vars, err := envfile.Read(strings.NewReader(tc.contents))
		if err == nil {
			t.Errorf("%q: expected an error", tc.contents)
			continue
		}
		if !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("%q: error %q does not contain %q", tc.contents, err.Error(), tc.errMsg)
		}
		if vars != nil {
			t.Errorf("%q: got %v, want a nil Variables", tc.contents, vars)
		}
	}
}

func TestVariablesLookup(t *testing.T) {
	vars, err := envfile.Read(strings.NewReader(sampleEnvfile))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// Lookup distinguishes a variable defined as empty from one that is not
	// defined at all; Mapping returns "" for both.
	if val, ok := vars.Lookup("EMPTY"); !ok || val != "" {
		t.Errorf("got %q, %v, want %q, true", val, ok, "")
	}
	if val, ok := vars.Lookup("MISSING"); ok || val != "" {
		t.Errorf("got %q, %v, want %q, false", val, ok, "")
	}
	if got := vars.Mapping("MISSING"); got != "" {
		t.Errorf("got %q, want an empty string", got)
	}
}

func TestVariablesExpand(t *testing.T) {
	vars := envfile.NewVariables(env("REGION", "us-west-2", "BUCKET", "app-logs", "TMPL", "$BUCKET/$REGION"))
	for _, tc := range []struct {
		input, want string
	}{
		{"", ""},
		{"no references", "no references"},
		{"$REGION", "us-west-2"},
		{"${REGION}", "us-west-2"},
		{"https://$REGION.example.com/$BUCKET", "https://us-west-2.example.com/app-logs"},
		// An undefined variable expands to "", as it does for os.Expand.
		{"[$MISSING]", "[]"},
		// Expansion is a single pass: references within a substituted value
		// are not themselves expanded.
		{"$TMPL", "$BUCKET/$REGION"},
	} {
		if got := vars.Expand(tc.input); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.input, got, tc.want)
		}
		// Expand is defined to be os.Expand with Mapping.
		if got := os.Expand(tc.input, vars.Mapping); got != tc.want {
			t.Errorf("%q: os.Expand: got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNewVariables(t *testing.T) {
	src := env("A", "1")
	vars := envfile.NewVariables(src)

	// The supplied map is copied, so later changes to it are not visible.
	src["A"] = "changed"
	src["B"] = "2"
	if got, want := vars.Mapping("A"), "1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, ok := vars.Lookup("B"); ok {
		t.Error("B should not be defined")
	}

	if got := envfile.NewVariables(nil).Len(); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestVariablesZeroAndNil(t *testing.T) {
	// The read-only methods treat both a nil and a zero-value Variables as
	// one with no definitions.
	for name, vars := range map[string]*envfile.Variables{
		"nil":        nil,
		"zero value": {},
	} {
		if got := vars.Len(); got != 0 {
			t.Errorf("%v: got %v, want 0", name, got)
		}
		if got := vars.Mapping("ANY"); got != "" {
			t.Errorf("%v: got %q, want an empty string", name, got)
		}
		if val, ok := vars.Lookup("ANY"); ok || val != "" {
			t.Errorf("%v: got %q, %v, want %q, false", name, val, ok, "")
		}
		if got := vars.Expand("$ANY-"); got != "-" {
			t.Errorf("%v: got %q, want %q", name, got, "-")
		}
		if got := vars.All(); len(got) != 0 {
			t.Errorf("%v: got %v, want no definitions", name, got)
		}
	}
}

func TestVariablesMerge(t *testing.T) {
	// Merging into a zero value works; its map is created on demand.
	vars := &envfile.Variables{}
	if err := vars.Merge(strings.NewReader("A=1\nB=2\n")); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// Later definitions overwrite earlier ones for the same name, and names
	// only in the earlier set are retained.
	if err := vars.Merge(strings.NewReader("B=two\nC=3\n")); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	want := env("A", "1", "B", "two", "C", "3")
	if got := vars.All(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if err := vars.MergeFile(writeEnvfile(t, "C=three\nD=4\n")); err != nil {
		t.Fatalf("MergeFile: %v", err)
	}
	want = env("A", "1", "B", "two", "C", "three", "D", "4")
	if got := vars.All(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestVariablesMergeErrors(t *testing.T) {
	vars := envfile.NewVariables(env("A", "1"))
	want := vars.All()

	// A failed merge leaves the existing definitions unchanged.
	if err := vars.Merge(strings.NewReader("A=2\nNOT A NAME=3\n")); err == nil {
		t.Error("expected an error")
	}
	if got := vars.All(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if err := vars.MergeFile(filepath.Join(t.TempDir(), "no-such-file")); err == nil {
		t.Error("expected an error for a file that does not exist")
	}
	if got := vars.All(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func ExampleRead() {
	const contents = `
# deployment settings
export REGION=us-west-2
BUCKET='app-logs'   # single quoted values are literal
TMPL="$BUCKET/$REGION"
`
	vars, err := envfile.Read(strings.NewReader(contents))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(vars.Len())
	fmt.Println(vars.Mapping("REGION"))
	if _, ok := vars.Lookup("MISSING"); !ok {
		fmt.Println("MISSING is not defined")
	}

	// Reading an envfile does not expand the references within its own
	// values; Mapping and Expand do.
	fmt.Println(vars.Mapping("TMPL"))
	fmt.Println(os.Expand(vars.Mapping("TMPL"), vars.Mapping))
	fmt.Println(vars.Expand("https://$REGION.example.com/${BUCKET}"))

	// Output:
	// 3
	// us-west-2
	// MISSING is not defined
	// $BUCKET/$REGION
	// app-logs/us-west-2
	// https://us-west-2.example.com/app-logs
}
