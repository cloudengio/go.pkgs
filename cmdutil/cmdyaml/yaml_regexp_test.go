// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

func TestRegexpUnmarshal(t *testing.T) {
	type cfg struct {
		Pattern cmdyaml.Regexp `yaml:"pattern"`
	}

	var c cfg
	if err := yaml.Unmarshal([]byte(`pattern: "^abc.*xyz$"`), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if c.Pattern.Regexp == nil {
		t.Fatal("Pattern.Regexp is nil")
	}
	if !c.Pattern.MatchString("abc123xyz") {
		t.Errorf("expected pattern to match %q", "abc123xyz")
	}
	if c.Pattern.MatchString("nope") {
		t.Errorf("expected pattern not to match %q", "nope")
	}
}

func TestRegexpUnmarshalInvalid(t *testing.T) {
	type cfg struct {
		Pattern cmdyaml.Regexp `yaml:"pattern"`
	}
	var c cfg
	if err := yaml.Unmarshal([]byte(`pattern: "("`), &c); err == nil {
		t.Fatal("expected an error for an invalid regular expression")
	}
}

func TestRegexpMarshalRoundTrip(t *testing.T) {
	type cfg struct {
		Pattern cmdyaml.Regexp `yaml:"pattern"`
	}

	var c cfg
	if err := yaml.Unmarshal([]byte(`pattern: "^[a-z]+$"`), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	out, err := yaml.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), "^[a-z]+$") {
		t.Errorf("marshaled output %q does not contain the pattern", out)
	}

	var c2 cfg
	if err := yaml.Unmarshal(out, &c2); err != nil {
		t.Fatalf("Unmarshal(round-trip): %v", err)
	}
	if c2.Pattern.String() != c.Pattern.String() {
		t.Errorf("round-trip: got %q, want %q", c2.Pattern.String(), c.Pattern.String())
	}
}

func TestRegexpZeroValue(t *testing.T) {
	var r cmdyaml.Regexp
	if got, want := r.String(), ""; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	out, err := yaml.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := strings.TrimSpace(string(out)), `""`; got != want {
		t.Errorf("Marshal(zero value) = %q, want %q", got, want)
	}
}

func TestRegexpListUnmarshal(t *testing.T) {
	type cfg struct {
		Patterns cmdyaml.RegexpList `yaml:"patterns"`
	}

	var c cfg
	in := "patterns: [\"^foo\", \"bar$\"]"
	if err := yaml.Unmarshal([]byte(in), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := len(c.Patterns), 2; got != want {
		t.Fatalf("got %v patterns, want %v: %v", got, want, c.Patterns)
	}
	if !c.Patterns[0].MatchString("foobar") {
		t.Errorf("expected patterns[0] to match %q", "foobar")
	}
	if !c.Patterns[1].MatchString("foobar") {
		t.Errorf("expected patterns[1] to match %q", "foobar")
	}
	if c.Patterns[0].MatchString("xfoo") {
		t.Errorf("expected patterns[0] (anchored) not to match %q", "xfoo")
	}
}

func TestRegexpListUnmarshalInvalid(t *testing.T) {
	type cfg struct {
		Patterns cmdyaml.RegexpList `yaml:"patterns"`
	}
	var c cfg
	if err := yaml.Unmarshal([]byte(`patterns: ["("]`), &c); err == nil {
		t.Fatal("expected an error for an invalid regular expression")
	}
}

func TestRegexpListMarshalRoundTrip(t *testing.T) {
	type cfg struct {
		Patterns cmdyaml.RegexpList `yaml:"patterns"`
	}

	var c cfg
	in := `patterns: ["^foo", "bar$"]`
	if err := yaml.Unmarshal([]byte(in), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	out, err := yaml.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{"^foo", "bar$"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("marshaled output %q does not contain %q", out, want)
		}
	}

	var c2 cfg
	if err := yaml.Unmarshal(out, &c2); err != nil {
		t.Fatalf("Unmarshal(round-trip): %v", err)
	}
	if got, want := len(c2.Patterns), len(c.Patterns); got != want {
		t.Fatalf("round-trip: got %v patterns, want %v", got, want)
	}
	for i := range c.Patterns {
		if c2.Patterns[i].String() != c.Patterns[i].String() {
			t.Errorf("round-trip[%d]: got %q, want %q", i, c2.Patterns[i].String(), c.Patterns[i].String())
		}
	}
}

func TestRegexpListEmpty(t *testing.T) {
	type cfg struct {
		Patterns cmdyaml.RegexpList `yaml:"patterns"`
	}
	var c cfg
	if err := yaml.Unmarshal([]byte(`patterns: []`), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := len(c.Patterns), 0; got != want {
		t.Errorf("got %v patterns, want %v", got, want)
	}
}
