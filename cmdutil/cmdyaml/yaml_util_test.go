// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"context"
	"os"
	"path/filepath"
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
		err := cmdyaml.ParseConfigString(tc.input, &ts)
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

func TestStrictParse(t *testing.T) {
	var ts testStruct
	input := `
field: [1,2]
unknown: [3,4]
`
	err := cmdyaml.ParseConfigString(input, &ts)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = cmdyaml.ParseConfigStringStrict(input, &ts)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if got, want := err.Error(), `line 3: "unknown: [3,4]": field unknown not found in type cmdyaml_test.testStruct`; !strings.Contains(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
