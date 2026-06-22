// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
)

type testStruct struct {
	Field []int
}

func TestYAMLErrors(t *testing.T) {
	var ts testStruct
	for i, tc := range []struct {
		input, errMsg string
	}{
		{`xxx: - err`, "yaml: block sequence entries are not allowed in this context"},

		{`
xxx: - err
`, `yaml: line 2: "xxx: - err": block sequence entries are not allowed in this context`},

		{`
	tab: 2`, `yaml: line 2: "\ttab: 2": found character that cannot start any token`},

		{`notab: 2
	tab: 3`, `yaml: line 2: "\ttab: 3": found a tab character that violates indentation`},

		{`	notab: 2`, `yaml: found character that cannot start any token`},

		{`

	tab: 2`, `yaml: line 3: "\ttab: 2": found character that cannot start any token`},

		{`
field:
  ts1: [1,2]`, "yaml: unmarshal errors:\n" + `  line 3: "  ts1: [1,2]": cannot unmarshal !!map into []int`},

		// Note that the yaml parser does not always get the line number correct!
		// It seems to be wrong for lists in particular.
		{`
list:
  - a
	  - b
`, `yaml: line 3: "  - a": found a tab character that violates indentation`},
	} {
		err := cmdyaml.NewParser().Parse(&ts, []byte(tc.input))
		if err == nil || strings.TrimSpace(err.Error()) != tc.errMsg {
			t.Errorf("%v: got %v, want %v", i, err, tc.errMsg)
		}
	}
}

type mergeStruct struct {
	A string
	B string
	C string
}

func writeTempYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestParseConfigFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := writeTempYAML(t, dir, "f1.yaml", "a: first\nb: from-f1\n")
	f2 := writeTempYAML(t, dir, "f2.yaml", "b: from-f2\nc: second\n")

	var cfg mergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg, f1, f2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// f1 sets A and B; f2 overrides B and adds C.
	if cfg.A != "first" {
		t.Errorf("A: got %q, want %q", cfg.A, "first")
	}
	if cfg.B != "from-f2" {
		t.Errorf("B: got %q, want %q", cfg.B, "from-f2")
	}
	if cfg.C != "second" {
		t.Errorf("C: got %q, want %q", cfg.C, "second")
	}
}

func TestParseConfigFilesNoFiles(t *testing.T) {
	var cfg mergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg); err == nil {
		t.Fatal("expected error for no files, got nil")
	}
}

func TestParseConfigFilesMissing(t *testing.T) {
	var cfg mergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg, "/no/such/file.yaml"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

type EmbeddedBase struct {
	X string
	Y string
}

// yaml.v3 does NOT inline embedded struct fields automatically (unlike encoding/json).
// Without a yaml:",inline" tag the embedded type's fields are nested under the
// lowercased type name (e.g. "embeddedbase:").
type embeddedMergeStruct struct {
	Top          string
	EmbeddedBase `yaml:",inline"`
}

func TestParseConfigFilesEmbeddedMerge(t *testing.T) {
	dir := t.TempDir()
	// Without yaml:",inline", the YAML key would be "embeddedbase:" not promoted.
	// With yaml:",inline" (as declared above) X and Y appear at the top level.
	// f1 sets Top, X, and Y; f2 overrides only X.
	f1 := writeTempYAML(t, dir, "f1.yaml", "top: from-f1\nx: x-from-f1\ny: y-from-f1\n")
	f2 := writeTempYAML(t, dir, "f2.yaml", "x: x-from-f2\n")

	var cfg embeddedMergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg, f1, f2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Top != "from-f1" {
		t.Errorf("Top: got %q, want %q", cfg.Top, "from-f1")
	}
	if cfg.X != "x-from-f2" {
		t.Errorf("X: got %q, want %q", cfg.X, "x-from-f2")
	}
	// Y is not mentioned in f2 and must survive from f1.
	if cfg.Y != "y-from-f1" {
		t.Errorf("Y: got %q, want %q", cfg.Y, "y-from-f1")
	}
}

type nestedMergeStruct struct {
	Top    string
	Nested struct {
		X string
		Y string
	}
	Items []string
}

func TestParseConfigFilesDeepMerge(t *testing.T) {
	dir := t.TempDir()
	// f1 sets Top, Nested.X, Nested.Y, and Items.
	f1 := writeTempYAML(t, dir, "f1.yaml", "top: from-f1\nnested:\n  x: x-from-f1\n  y: y-from-f1\nitems:\n  - a\n  - b\n")
	// f2 overrides only Nested.X and Items; Top and Nested.Y should survive from f1.
	f2 := writeTempYAML(t, dir, "f2.yaml", "nested:\n  x: x-from-f2\nitems:\n  - c\n")

	var cfg nestedMergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg, f1, f2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Top != "from-f1" {
		t.Errorf("Top: got %q, want %q", cfg.Top, "from-f1")
	}
	if cfg.Nested.X != "x-from-f2" {
		t.Errorf("Nested.X: got %q, want %q", cfg.Nested.X, "x-from-f2")
	}
	// Deep struct merge: Nested.Y is not mentioned in f2 and must survive from f1.
	if cfg.Nested.Y != "y-from-f1" {
		t.Errorf("Nested.Y: got %q, want %q", cfg.Nested.Y, "y-from-f1")
	}
	// Slices are replaced, not merged: f2's Items replaces f1's Items entirely.
	if len(cfg.Items) != 1 || cfg.Items[0] != "c" {
		t.Errorf("Items: got %v, want [c]", cfg.Items)
	}
}

func TestParseConfigFilesWithAnchors(t *testing.T) {
	dir := t.TempDir()
	f1 := writeTempYAML(t, dir, "f1.yaml", "_defaults: &defaults\n  b: from-anchor\n")
	f2 := writeTempYAML(t, dir, "f2.yaml", "a: hello\n<<: *defaults\n")

	var cfg mergeStruct
	if err := cmdyaml.ParseConfigFiles(context.Background(), &cfg, f1, f2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.A != "hello" {
		t.Errorf("A: got %q, want %q", cfg.A, "hello")
	}
	if cfg.B != "from-anchor" {
		t.Errorf("B: got %q, want %q", cfg.B, "from-anchor")
	}
}

func TestParseConfigFilesWithAnchorAndError(t *testing.T) {
	dir := t.TempDir()
	f1 := writeTempYAML(t, dir, "f1.yaml", "_defaults: &defaults\n  b: from-anchor\n")
	// f2 has an invalid field 'd' which should cause a strict parsing error.
	f2 := writeTempYAML(t, dir, "f2.yaml", "a: hello\n<<: *defaults\nd: error\n")

	var cfg mergeStruct
	err := cmdyaml.ParseConfigFilesStrict(context.Background(), &cfg, f1, f2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error should point to line 3 of f2.yaml.
	want := fmt.Sprintf("%s: line 3:", f2)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error message %q does not contain %q", err.Error(), want)
	}
}

func TestParseConfigFilesStrict(t *testing.T) {
	dir := t.TempDir()
	f1 := writeTempYAML(t, dir, "ok.yaml", "a: hello\n")
	fBad := writeTempYAML(t, dir, "bad.yaml", "a: hello\nunknown: field\n")

	var cfg mergeStruct
	if err := cmdyaml.ParseConfigFilesStrict(context.Background(), &cfg, f1); err != nil {
		t.Fatalf("strict single file: unexpected error: %v", err)
	}
	if err := cmdyaml.ParseConfigFilesStrict(context.Background(), &cfg, fBad); err == nil {
		t.Fatal("strict: expected error for unknown field, got nil")
	}
}

func TestParseConfigs(t *testing.T) {
	s1 := []byte("a: first\nb: from-s1\n")
	s2 := []byte("b: from-s2\nc: second\n")

	var cfg mergeStruct
	if err := cmdyaml.ParseConfigs(&cfg, s1, s2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.A != "first" {
		t.Errorf("A: got %q, want %q", cfg.A, "first")
	}
	if cfg.B != "from-s2" {
		t.Errorf("B: got %q, want %q", cfg.B, "from-s2")
	}
	if cfg.C != "second" {
		t.Errorf("C: got %q, want %q", cfg.C, "second")
	}
}

func TestParseConfigsNoSpecs(t *testing.T) {
	var cfg mergeStruct
	// Zero specs is an error.
	if err := cmdyaml.ParseConfigs(&cfg); err == nil {
		t.Fatalf("expected error for zero specs, got nil")
	}
}

func TestParseConfigsStrict(t *testing.T) {
	var cfg mergeStruct
	if err := cmdyaml.ParseConfigsStrict(&cfg, []byte("a: hello\n")); err != nil {
		t.Fatalf("strict single spec: unexpected error: %v", err)
	}
	if err := cmdyaml.ParseConfigsStrict(&cfg, []byte("a: hello\nunknown: field\n")); err == nil {
		t.Fatal("strict: expected error for unknown field, got nil")
	}
}

// TestCrossSpecAnchorOrdering verifies that when one anchor's value references
// another anchor via *alias, the preamble injected for later specs emits
// anchor definitions in dependency order regardless of map-iteration ordering.
// spec1 defines _base and _ext where _ext merges *base; spec2 references *ext.
// The preamble for spec2 must emit _base before _ext so that *base inside
// _ext's serialised value resolves correctly. Running 100 iterations exposes
// non-determinism, because Go randomises map iteration on every pass.
func TestCrossSpecAnchorOrdering(t *testing.T) {
	type result struct {
		X int
		Y int
	}
	spec1 := []byte(`
_base: &base
  x: 1
_ext: &ext
  <<: *base
  y: 2
`)
	spec2 := []byte(`<<: *ext`)
	for range 100 {
		var cfg result
		if err := cmdyaml.ParseConfigs(&cfg, spec1, spec2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.X != 1 {
			t.Errorf("X: got %d, want 1", cfg.X)
		}
		if cfg.Y != 2 {
			t.Errorf("Y: got %d, want 2", cfg.Y)
		}
	}
}

// TestParseConfigStrictAnchorAllowed verifies that a top-level field whose
// value carries a YAML anchor does not trigger a strict-mode unknown-field
// error. The anchor is consumed via a merge key (<<) so its content appears
// in the decoded struct.
func TestParseConfigStrictAnchorAllowed(t *testing.T) {
	const input = `
_defaults: &defaults
  b: from-anchor

a: hello
<<: *defaults
`
	var cfg mergeStruct
	if err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.A != "hello" {
		t.Errorf("A: got %q, want %q", cfg.A, "hello")
	}
	if cfg.B != "from-anchor" {
		t.Errorf("B: got %q, want %q", cfg.B, "from-anchor")
	}
}

// TestParseConfigStrictNestedAnchorAllowed verifies that an anchor-definition
// field nested inside a struct field (not at the top level) is also permitted.
func TestParseConfigStrictNestedAnchorAllowed(t *testing.T) {
	// _sub_defaults is inside the "nested:" block, not at the top level.
	// allAnchorFields must walk into nested mappings to find it.
	const input = `
top: hello
nested:
  _sub_defaults: &sub_defaults
    y: from-nested-anchor
  x: direct
  <<: *sub_defaults
`
	var cfg nestedMergeStruct
	if err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Top != "hello" {
		t.Errorf("Top: got %q, want %q", cfg.Top, "hello")
	}
	if cfg.Nested.X != "direct" {
		t.Errorf("Nested.X: got %q, want %q", cfg.Nested.X, "direct")
	}
	if cfg.Nested.Y != "from-nested-anchor" {
		t.Errorf("Nested.Y: got %q, want %q", cfg.Nested.Y, "from-nested-anchor")
	}
}

// TestParseConfigStrictAnchorInSequenceAllowed verifies that an anchor-definition
// field inside a sequence element mapping is also permitted.
func TestParseConfigStrictAnchorInSequenceAllowed(t *testing.T) {
	// _base is inside a sequence item mapping.
	type seqStruct struct {
		Items []struct {
			Name  string
			Value string
		}
	}
	const input = `
items:
  - _base: &base
      value: shared
    name: first
    <<: *base
  - name: second
    value: direct
`
	var cfg seqStruct
	if err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Items) != 2 {
		t.Fatalf("Items: got %d, want 2", len(cfg.Items))
	}
	if cfg.Items[0].Name != "first" {
		t.Errorf("Items[0].Name: got %q, want %q", cfg.Items[0].Name, "first")
	}
	if cfg.Items[0].Value != "shared" {
		t.Errorf("Items[0].Value: got %q, want %q", cfg.Items[0].Value, "shared")
	}
	if cfg.Items[1].Value != "direct" {
		t.Errorf("Items[1].Value: got %q, want %q", cfg.Items[1].Value, "direct")
	}
}

// TestParseConfigStrictAnchorPlusUnknown verifies that a genuinely unknown
// field is still rejected even when an anchor-definition field is also present.
func TestParseConfigStrictAnchorPlusUnknown(t *testing.T) {
	const input = `
_defaults: &defaults
  b: from-anchor

unknown_field: bad
a: hello
`
	var cfg mergeStruct
	if err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input)); err == nil {
		t.Fatal("expected error for unknown_field, got nil")
	}
}

// TestParseConfigStrictAnchorUnknownNoPanic verifies that an unknown field
// error in the anchor-expansion path does not panic. Before the fix,
// ErrorWithSource was called with spec but error line numbers referred to the
// re-marshalled cleaned form, which can have more lines than spec (keys are
// sorted alphabetically and flow-style values are expanded), causing an
// out-of-bounds index into specLines.
func TestParseConfigStrictAnchorUnknownNoPanic(t *testing.T) {
	// The anchor definition field (_d) is stripped before strict decoding.
	// cleaned = yaml.Marshal({"a":"hello","unknown":"bad"}) which sorts keys
	// and may differ in line count from the original spec.
	// An unknown field in cleaned must not panic ErrorWithSource.
	const input = `
_d: &d
  b: from-anchor

a: hello
unknown: bad
`
	var cfg mergeStruct
	err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input))
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	// Confirm it did not panic and the error is non-empty.
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

// TestErrorLineNumber_StrictUnknownField verifies that the line number and
// source content in a strict-mode unknown-field error accurately reflect the
// position and text of the offending field in the original spec.
func TestErrorLineNumber_StrictUnknownField(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input       string
		wantLine    int
		wantContent string
	}{
		{
			name:        "unknown field on line 1",
			input:       "unknown: bad\na: hello\n",
			wantLine:    1,
			wantContent: "unknown: bad",
		},
		{
			name:        "unknown field on line 2",
			input:       "a: hello\nunknown: bad\n",
			wantLine:    2,
			wantContent: "unknown: bad",
		},
		{
			name:        "unknown field after blank line",
			input:       "a: hello\n\nunknown: bad\n",
			wantLine:    3,
			wantContent: "unknown: bad",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cfg mergeStruct
			err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(tc.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			msg := err.Error()
			wantPrefix := fmt.Sprintf("line %d:", tc.wantLine)
			if !strings.Contains(msg, wantPrefix) {
				t.Errorf("error %q does not contain %q", msg, wantPrefix)
			}
			if !strings.Contains(msg, tc.wantContent) {
				t.Errorf("error %q does not contain source content %q", msg, tc.wantContent)
			}
		})
	}
}

// TestErrorLineNumber_MultipleUnknownFields verifies that each unknown-field
// error carries an accurate line number and source content when multiple
// unknown fields are present in a single spec.
func TestErrorLineNumber_MultipleUnknownFields(t *testing.T) {
	const input = "a: hello\nunknown1: bad\nb: world\nunknown2: also-bad\n"
	var cfg mergeStruct
	err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{
		`line 2:`, `unknown1: bad`,
		`line 4:`, `unknown2: also-bad`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not contain %q", msg, want)
		}
	}
}

// TestErrorLineNumber_TypeMismatch verifies that a type-mismatch error
// reports the correct line and shows the offending source content.
func TestErrorLineNumber_TypeMismatch(t *testing.T) {
	// testStruct.Field expects []int; a map value triggers a TypeError.
	const input = "field:\n  sub: not-an-int\n"
	var ts testStruct
	err := cmdyaml.NewParser().Parse(&ts, []byte(input))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line 2:") {
		t.Errorf("error %q does not contain \"line 2:\"", msg)
	}
	if !strings.Contains(msg, "sub: not-an-int") {
		t.Errorf("error %q does not contain the offending source line", msg)
	}
}

// TestErrorLineNumber_AnchorPathAccuracy verifies error-source accuracy when
// the anchor-expansion path is taken. Because cleaned is re-marshalled (keys
// sorted, flow nodes expanded) its line numbers differ from spec. The error
// must reference content from cleaned (not spec), must name the unknown field,
// and must not panic.
func TestErrorLineNumber_AnchorPathAccuracy(t *testing.T) {
	// 'unknown' appears on line 3 of spec (after the anchor block) but on
	// line 1 of cleaned (keys are sorted: 'a' < 'unknown' is false here —
	// 'a' sorts before 'u', so cleaned = "a: hello\nunknown: bad\n").
	// Either way, ErrorWithSource uses cleaned, so the source context shown
	// in the error must come from cleaned, not spec.
	const input = `unknown: bad
_d: &d
  b: from-anchor
a: hello
`
	var cfg mergeStruct
	err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input))
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	msg := err.Error()
	// The unknown field name must appear in the error.
	if !strings.Contains(msg, "unknown") {
		t.Errorf("error %q should name the unknown field", msg)
	}
	// The source content in the error comes from cleaned, where the line text
	// for the unknown field is always "unknown: bad".
	if !strings.Contains(msg, "unknown: bad") {
		t.Errorf("error %q should include offending field text from cleaned form", msg)
	}
}

// TestErrorLineNumber_FlowExpansionNoPanic verifies that when a non-anchor
// field contains a flow-style map that gets expanded to multiple block lines
// in cleaned, ErrorWithSource does not panic even though cleaned may have
// more lines than spec.
func TestErrorLineNumber_FlowExpansionNoPanic(t *testing.T) {
	// After stripping _d, cleaned = yaml.Marshal({"a":"hello",
	// "b":{"p":6,"q":7,"v":1,"w":2,"x":3,"y":4,"z":5}, "unknown":"bad"}).
	// That expands b's flow map into 7 sub-lines, pushing 'unknown' to a
	// line beyond the total line count of spec. Before the fix this caused
	// a panic; after the fix ErrorWithSource uses cleaned, so lines always
	// stay in range.
	const input = `_d: &d
  ignored: value
a: hello
b: {p: 6, q: 7, v: 1, w: 2, x: 3, y: 4, z: 5}
unknown: bad
`
	var cfg mergeStruct
	err := cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&cfg, []byte(input))
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error %q should mention the unknown field", err.Error())
	}
}

// TestStrictVariablesMapAllowed verifies that the variables map name is not
// treated as an unknown field when strict mode and WithYAMLVariables are both
// active.
func TestStrictVariablesMapAllowed(t *testing.T) {
	type config struct {
		Host  string            `yaml:"host"`
		Dummy map[string]string `yaml:"dummy"`
	}
	spec := []byte(`
dummy:
  ignore: me
vars:
  greeting: hello
host: ${greeting}
`)
	var cfg config
	p := cmdyaml.NewParser(
		cmdyaml.WithStrictFields(true),
		cmdyaml.WithYAMLVariables("vars"),
	)
	if err := p.Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error with variables map in strict mode: %v", err)
	}
	if cfg.Host != "hello" {
		t.Errorf("Host: got %q, want %q", cfg.Host, "hello")
	}
}

// TestStrictVariablesMapRealUnknownStillRejected verifies that a genuinely
// unknown field is still rejected even when WithYAMLVariables is set.
func TestStrictVariablesMapRealUnknownStillRejected(t *testing.T) {
	type config struct {
		Host string `yaml:"host"`
	}
	spec := []byte(`
vars:
  greeting: hello
host: world
truly_unknown: bad
`)
	var cfg config
	p := cmdyaml.NewParser(
		cmdyaml.WithStrictFields(true),
		cmdyaml.WithYAMLVariables("vars"),
	)
	if err := p.Parse(&cfg, spec); err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

// TestVariablesSingleSpecExpansion verifies that ${VAR} references in a spec
// are replaced by values from the vars block declared in the same spec.
func TestVariablesSingleSpecExpansion(t *testing.T) {
	type config struct {
		Host string `yaml:"host"`
		Addr string `yaml:"addr"`
	}
	spec := []byte(`
vars:
  h: localhost
  suffix: api
host: ${h}
addr: ${h}/${suffix}
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Host, "localhost"; got != want {
		t.Errorf("Host: got %q, want %q", got, want)
	}
	if got, want := cfg.Addr, "localhost/api"; got != want {
		t.Errorf("Addr: got %q, want %q", got, want)
	}
}

// TestVariablesCrossSpecExpansion verifies that variables defined in an earlier
// spec's vars block are expanded when processing a later spec that has no vars
// block of its own.
func TestVariablesCrossSpecExpansion(t *testing.T) {
	type config struct {
		Host string `yaml:"host"`
		Path string `yaml:"path"`
	}
	spec1 := []byte(`
vars:
  base: myserver
`)
	spec2 := []byte(`
host: ${base}
path: /api/${base}
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).Parse(&cfg, spec1, spec2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Host, "myserver"; got != want {
		t.Errorf("Host: got %q, want %q", got, want)
	}
	if got, want := cfg.Path, "/api/myserver"; got != want {
		t.Errorf("Path: got %q, want %q", got, want)
	}
}

// TestVariablesAccumulateAcrossSpecs verifies that each spec's vars block adds
// to the accumulated variable set, so a later spec can reference variables from
// both its own vars block and those of earlier specs.
func TestVariablesAccumulateAcrossSpecs(t *testing.T) {
	type config struct {
		A string `yaml:"a"`
		B string `yaml:"b"`
	}
	spec1 := []byte(`
vars:
  x: hello
a: ${x}
`)
	spec2 := []byte(`
vars:
  y: world
b: ${x}-${y}
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).Parse(&cfg, spec1, spec2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.A, "hello"; got != want {
		t.Errorf("A: got %q, want %q", got, want)
	}
	if got, want := cfg.B, "hello-world"; got != want {
		t.Errorf("B: got %q, want %q", got, want)
	}
}

func TestVariablesExpansionInVars(t *testing.T) {
	type config struct {
		BA string `yaml:"ba"`
		A  string `yaml:"a"`
		B  string `yaml:"b"`
		C  string `yaml:"c"`
	}
	spec1 := []byte(`
vars:
  ba: ${a}
  a: hello
  x: ${a}
ba: ${ba}
a: ${x}
`)
	spec2 := []byte(`
vars:
  y: world
  z: ${y}
b: ${x}-${y}
`)
	spec3 := []byte(`
c: ${x}-${y}-${z}
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).Parse(&cfg, spec1, spec2, spec3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.BA, "hello"; got != want {
		t.Errorf("BA: got %q, want %q", got, want)
	}
	if got, want := cfg.A, "hello"; got != want {
		t.Errorf("A: got %q, want %q", got, want)
	}
	if got, want := cfg.B, "hello-world"; got != want {
		t.Errorf("B: got %q, want %q", got, want)
	}
	if got, want := cfg.C, "hello-world-world"; got != want {
		t.Errorf("C: got %q, want %q", got, want)
	}
}

// TestVariablesLaterSpecOverrides verifies that a variable redefined in a later
// spec's vars block overrides the value from the earlier spec.
func TestVariablesLaterSpecOverrides(t *testing.T) {
	type config struct {
		Host string `yaml:"host"`
	}
	spec1 := []byte(`
vars:
  target: old-host
`)
	spec2 := []byte(`
vars:
  target: new-host
host: ${target}
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).Parse(&cfg, spec1, spec2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Host, "new-host"; got != want {
		t.Errorf("Host: got %q, want %q", got, want)
	}
}

// TestVariablesExpansionFiles verifies that ParseFiles accumulates variables
// across files: a variable defined in file 1's vars block expands references
// found in file 2.
func TestVariablesExpansionFiles(t *testing.T) {
	type config struct {
		Endpoint string `yaml:"endpoint"`
	}
	dir := t.TempDir()
	f1 := writeTempYAML(t, dir, "f1.yaml", "vars:\n  host: myserver\n  port: nine090\n")
	f2 := writeTempYAML(t, dir, "f2.yaml", "endpoint: http://${host}:${port}/api\n")

	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithYAMLVariables("vars")).ParseFiles(context.Background(), &cfg, f1, f2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Endpoint, "http://myserver:nine090/api"; got != want {
		t.Errorf("Endpoint: got %q, want %q", got, want)
	}
}

// TestWithExpandMapping verifies that WithExpandMapping expands $VAR and ${VAR}
// references in the raw spec bytes using the provided function before parsing.
func TestWithExpandMapping(t *testing.T) {
	type config struct {
		Addr string `yaml:"addr"`
		Name string `yaml:"name"`
	}
	spec := []byte(`
addr: ${HOST}
name: $APP_NAME
`)
	envFn := func(key string) string {
		return map[string]string{"HOST": "localhost", "APP_NAME": "myapp"}[key]
	}
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithExpandMapping(envFn)).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Addr, "localhost"; got != want {
		t.Errorf("Addr: got %q, want %q", got, want)
	}
	if got, want := cfg.Name, "myapp"; got != want {
		t.Errorf("Name: got %q, want %q", got, want)
	}
}

// TestVariablesExpandMappingOrdering verifies that when both WithYAMLVariables
// and WithExpandMapping are active. This means that env variables can be used
// in the vars block itself, and that variables from the vars block can be used
// in the same spec alongside env variables, regardless of map iteration order.
func TestVariablesExpandMappingOrdering(t *testing.T) {
	type config struct {
		Addr    string `yaml:"addr"`
		EnvAddr string `yaml:"env_addr"`
		Tag     string `yaml:"tag"`
	}
	spec := []byte(`
vars:
  host: myserver
  envhost: ${env_host}
addr: ${host}
env_addr: ${envhost}
tag: ${env_tag}
`)
	envFn := func(key string) string {
		return map[string]string{
			"env_tag":  "prod",
			"env_host": "another-server",
		}[key]
	}
	var cfg config
	p := cmdyaml.NewParser(
		cmdyaml.WithYAMLVariables("vars"),
		cmdyaml.WithExpandMapping(envFn),
	)
	if err := p.Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Addr, "myserver"; got != want {
		t.Errorf("Addr: got %q, want %q", got, want)
	}

	if got, want := cfg.Tag, "prod"; got != want {
		t.Errorf("Tag: got %q, want %q", got, want)
	}
	if got, want := cfg.EnvAddr, "another-server"; got != want {
		t.Errorf("EnvAddr: got %q, want %q", got, want)
	}
}

// TestVarsAndAnchorReferenceInSameFile verifies that a spec can simultaneously
// carry a vars block and reference an anchor defined in an earlier spec.
// This exercises the case where expandSpec must resolve cross-file anchors
// before it can successfully unmarshal the vars block.
func TestVarsAndAnchorReferenceInSameFile(t *testing.T) {
	type config struct {
		Name    string `yaml:"name"`
		Host    string `yaml:"host"`
		Version string `yaml:"version"`
	}

	// File 1: anchor definition only, no vars block.
	f1Content := "_service: &service\n  host: svc.internal\n  version: v1\n"
	// File 2: vars block whose name expands ${name}, PLUS a merge key that
	// references the anchor from file 1.  Without the fix, variables.Load
	// fails with "unknown anchor 'service'" when it calls yaml.Unmarshal on
	// these raw bytes.
	f2Content := strings.Join([]string{
		"vars:",
		"  name: myservice",
		"name: ${name}",
		"<<: *service",
		"",
	}, "\n")

	for _, tc := range []struct {
		name   string
		strict bool
	}{
		{"non-strict", false},
		{"strict", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			f1 := writeTempYAML(t, dir, "base.yaml", f1Content)
			f2 := writeTempYAML(t, dir, "svc.yaml", f2Content)

			opts := []cmdyaml.Option{cmdyaml.WithYAMLVariables("vars")}
			if tc.strict {
				opts = append(opts, cmdyaml.WithStrictFields(true))
			}
			var cfg config
			if err := cmdyaml.NewParser(opts...).ParseFiles(context.Background(), &cfg, f1, f2); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := cfg.Name, "myservice"; got != want {
				t.Errorf("Name: got %q, want %q", got, want)
			}
			if got, want := cfg.Host, "svc.internal"; got != want {
				t.Errorf("Host: got %q, want %q", got, want)
			}
			if got, want := cfg.Version, "v1"; got != want {
				t.Errorf("Version: got %q, want %q", got, want)
			}
		})
	}
}

// TestVariablesAndAnchorsAcrossFiles verifies that variables defined in a vars
// block in file 1 are expanded into an anchor in the same file, and that anchor
// is correctly resolved when referenced via a merge key in file 2.  The test
// also runs in strict mode to confirm that the anchor-definition field and the
// vars map name are both permitted.
func TestVariablesAndAnchorsAcrossFiles(t *testing.T) {
	type config struct {
		Name string `yaml:"name"`
		Host string `yaml:"host"`
		Path string `yaml:"path"`
		Tag  string `yaml:"tag"`
	}

	// File 1: declares vars and an anchor whose values use those vars.
	// _conn is the anchor-definition field; its value carries &conn.
	f1Content := strings.Join([]string{
		"vars:",
		"  host: db.internal",
		"  base_path: /v1",
		"_conn: &conn",
		"  host: ${host}",
		"  path: ${base_path}",
		"",
	}, "\n")

	// File 2: merges the anchor from file 1 and adds its own fields.
	f2Content := strings.Join([]string{
		"name: payments",
		"tag: live",
		"<<: *conn",
		"",
	}, "\n")

	for _, tc := range []struct {
		name   string
		strict bool
	}{
		{"non-strict", false},
		{"strict", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			f1 := writeTempYAML(t, dir, "base.yaml", f1Content)
			f2 := writeTempYAML(t, dir, "svc.yaml", f2Content)

			opts := []cmdyaml.Option{cmdyaml.WithYAMLVariables("vars")}
			if tc.strict {
				opts = append(opts, cmdyaml.WithStrictFields(true))
			}
			var cfg config
			if err := cmdyaml.NewParser(opts...).ParseFiles(context.Background(), &cfg, f1, f2); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := cfg.Name, "payments"; got != want {
				t.Errorf("Name: got %q, want %q", got, want)
			}
			if got, want := cfg.Tag, "live"; got != want {
				t.Errorf("Tag: got %q, want %q", got, want)
			}
			// These two come from the anchor, which was expanded using the vars block.
			if got, want := cfg.Host, "db.internal"; got != want {
				t.Errorf("Host: got %q, want %q", got, want)
			}
			if got, want := cfg.Path, "/v1"; got != want {
				t.Errorf("Path: got %q, want %q", got, want)
			}
		})
	}
}

func TestStrictParse(t *testing.T) {
	var ts testStruct
	input := `
field: [1,2]
unknown: [3,4]
`
	err := cmdyaml.NewParser().Parse(&ts, []byte(input))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = cmdyaml.NewParser(cmdyaml.WithStrictFields(true)).Parse(&ts, []byte(input))
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if got, want := err.Error(), `line 3: "unknown: [3,4]": field unknown not found in type cmdyaml_test.testStruct`; !strings.Contains(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSequenceMergeNestedCrossSpec exposes the bug where a preamble anchor
// whose own content contains a sequence merge is not recursively flattened
// before being inlined. spec1 defines _inner and _outer (which itself merges
// *inner); spec2 merges *outer into items. Without the fix, items receives the
// un-flattened content of _outer — including the raw {<<: *inner} mapping node
// — instead of the fully resolved [a, b, c, d].
func TestSequenceMergeNestedCrossSpec(t *testing.T) {
	type config struct {
		Items []string `yaml:"items"`
	}
	spec1 := []byte(`
_inner: &inner
  - a
  - b
_outer: &outer
  - <<: *inner
  - c
`)
	spec2 := []byte(`
items:
  - <<: *outer
  - d
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec1, spec2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Items, []string{"a", "b", "c", "d"}; !slices.Equal(got, want) {
		t.Errorf("Items: got %v, want %v", got, want)
	}
}

// TestSequenceMergeNestedSameSpec verifies that nested sequence merges within
// a single spec are fully resolved (regression: this already worked via
// in-place modification of anchor nodes during the DFS walk, but must continue
// to work after adding cycle detection).
func TestSequenceMergeNestedSameSpec(t *testing.T) {
	type config struct {
		Items []string `yaml:"items"`
	}
	spec := []byte(`
_inner: &inner
  - a
  - b
_outer: &outer
  - <<: *inner
  - c
items:
  - <<: *outer
  - d
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Items, []string{"a", "b", "c", "d"}; !slices.Equal(got, want) {
		t.Errorf("Items: got %v, want %v", got, want)
	}
}

// TestSequenceMergeMultiDocSpec verifies that sequence merging preserves all
// YAML documents in a multi-document spec. Before the fix, applySequenceMerge
// took only docs[len(docs)-1], silently dropping every preceding spec document.
func TestSequenceMergeMultiDocSpec(t *testing.T) {
	type config struct {
		A []string `yaml:"a"`
		B []string `yaml:"b"`
	}
	// Two documents in one spec: first defines the anchor and field A,
	// second references the same anchor for field B.
	spec := []byte(`
_base: &base
  - shared

a:
  - <<: *base
  - only-a
---
b:
  - <<: *base
  - only-b
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.A, []string{"shared", "only-a"}; !slices.Equal(got, want) {
		t.Errorf("A: got %v, want %v", got, want)
	}
	if got, want := cfg.B, []string{"shared", "only-b"}; !slices.Equal(got, want) {
		t.Errorf("B: got %v, want %v", got, want)
	}
}

// TestSequenceMergeSameSpec verifies that <<: *anchor inside a sequence
// flattens the referenced sequence inline when anchor and reference are in
// the same spec.
func TestSequenceMergeSameSpec(t *testing.T) {
	type config struct {
		Items []string `yaml:"items"`
	}
	spec := []byte(`
_base: &base
  - a
  - b

items:
  - <<: *base
  - c
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Items, []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("Items: got %v, want %v", got, want)
	}
}

// TestSequenceMergeCrossSpec verifies that an anchor defined in one spec can
// be merged into a sequence in a later spec.
func TestSequenceMergeCrossSpec(t *testing.T) {
	type config struct {
		Items []string `yaml:"items"`
	}
	spec1 := []byte(`
_base: &base
  - x
  - y
`)
	spec2 := []byte(`
items:
  - <<: *base
  - z
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec1, spec2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := cfg.Items, []string{"x", "y", "z"}; !slices.Equal(got, want) {
		t.Errorf("Items: got %v, want %v", got, want)
	}
}

// TestSequenceMergeStrictMode verifies that the anchor-definition field and
// the merged sequence do not trigger strict-mode unknown-field errors.
func TestSequenceMergeStrictMode(t *testing.T) {
	type config struct {
		Items []string `yaml:"items"`
	}
	spec := []byte(`
_base: &base
  - p
  - q

items:
  - <<: *base
  - r
`)
	var cfg config
	p := cmdyaml.NewParser(
		cmdyaml.WithSequenceMerge("<<"),
		cmdyaml.WithStrictFields(true),
	)
	if err := p.Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected strict-mode error: %v", err)
	}
	if got, want := cfg.Items, []string{"p", "q", "r"}; !slices.Equal(got, want) {
		t.Errorf("Items: got %v, want %v", got, want)
	}
}

// TestSequenceMergeWithVariables verifies that sequence merging and variable
// expansion work together: variable values inside merged items are expanded.
func TestSequenceMergeWithVariables(t *testing.T) {
	type config struct {
		Routes []string `yaml:"routes"`
	}
	spec := []byte(`
vars:
  prefix: /api/v1

_base: &base
  - ${prefix}/users
  - ${prefix}/posts

routes:
  - <<: *base
  - ${prefix}/admin
`)
	var cfg config
	p := cmdyaml.NewParser(
		cmdyaml.WithSequenceMerge("<<"),
		cmdyaml.WithYAMLVariables("vars"),
	)
	if err := p.Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"/api/v1/users", "/api/v1/posts", "/api/v1/admin"}
	if got := cfg.Routes; !slices.Equal(got, want) {
		t.Errorf("Routes: got %v, want %v", got, want)
	}
}

// TestSequenceMergeMultiple verifies that multiple <<: *anchor elements in the
// same sequence are each flattened in order.
func TestSequenceMergeMultiple(t *testing.T) {
	type config struct {
		All []string `yaml:"all"`
	}
	spec := []byte(`
_a: &a
  - one
  - two
_b: &b
  - three
  - four

all:
  - <<: *a
  - <<: *b
  - five
`)
	var cfg config
	if err := cmdyaml.NewParser(cmdyaml.WithSequenceMerge("<<")).Parse(&cfg, spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"one", "two", "three", "four", "five"}
	if got := cfg.All; !slices.Equal(got, want) {
		t.Errorf("All: got %v, want %v", got, want)
	}
}
