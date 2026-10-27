// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdconfig"
	"cloudeng.io/cmdutil/cmdyaml"
	"cloudeng.io/cmdutil/envfile"
)

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("LOAD_TEST_FROM_ENV", "env-value")
	writeFile(t, dir, "shared.yaml", "name: ${LOAD_TEST_NAME}\nenvValue: ${LOAD_TEST_FROM_ENV}\n")
	writeFile(t, dir, "app.yaml", "extra: ${LOAD_TEST_EXTRA}\n")
	writeFile(t, dir, "one.env", "LOAD_TEST_NAME=first\nLOAD_TEST_EXTRA=from-one\nLOAD_TEST_FROM_ENV=file-loses\n")
	writeFile(t, dir, "two.env", "LOAD_TEST_NAME=second\n")

	type config struct {
		Name     string `yaml:"name"`
		EnvValue string `yaml:"envValue"`
		Extra    string `yaml:"extra"`
	}
	var ff cmdconfig.FilesFlag
	ff.SetDefaults("shared.yaml", "app.yaml", "one.env", "two.env", "absent.yaml")
	var cfg config
	loc, err := cmdconfig.Load(context.Background(), &cfg, &ff, cmdconfig.Spec{Path: cmdconfig.SearchPath("loadtest")})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loc.YAML, []string{"shared.yaml", "app.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// The later env file overrides the earlier, and the environment overrides both.
	want := config{Name: "second", EnvValue: "env-value", Extra: "from-one"}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "ok.yaml", "name: x\n")
	writeFile(t, dir, "unknown.yaml", "nope: x\n")
	writeFile(t, dir, "bad.env", "BAD NAME=x\n")
	type config struct {
		Name string `yaml:"name"`
	}
	spec := cmdconfig.Spec{Path: cmdconfig.SearchPath("loadtest")}
	load := func(files string) (*cmdconfig.Located, error) {
		var ff cmdconfig.FilesFlag
		if err := ff.Set(files); err != nil {
			t.Fatal(err)
		}
		var cfg config
		return cmdconfig.Load(context.Background(), &cfg, &ff, spec)
	}
	// A file that was set and does not exist.
	if _, err := load("missing.yaml"); err == nil || !strings.Contains(err.Error(), "missing.yaml") {
		t.Errorf("unexpected error: %v", err)
	}
	// Strict parsing.
	if loc, err := load("unknown.yaml"); err == nil || loc == nil || len(loc.YAML) != 1 {
		t.Errorf("unexpected: %v, %v", loc, err)
	}
	// An environment variables file that cannot be parsed.
	if _, err := load("ok.yaml,bad.env"); err == nil || !strings.Contains(err.Error(), "bad.env") {
		t.Errorf("unexpected error: %v", err)
	}
	// No YAML at all is not an error.
	var ff cmdconfig.FilesFlag
	var cfg config
	loc, err := cmdconfig.Load(context.Background(), &cfg, &ff, spec)
	if err != nil || len(loc.YAML) != 0 || cfg.Name != "" {
		t.Errorf("unexpected: %v, %v, %+v", loc, err, cfg)
	}
}

func TestLoadFiles(t *testing.T) {
	dir := t.TempDir()
	one := writeFile(t, dir, "one.yaml", "name: ${LOAD_TEST_NAME}\n")
	two := writeFile(t, dir, "two.yaml", "other: 2\n")
	vars := writeFile(t, dir, "vars.env", "LOAD_TEST_NAME=named\n")
	var cfg struct {
		Name  string `yaml:"name"`
		Other int    `yaml:"other"`
	}
	loc := &cmdconfig.Located{YAML: []string{one}, Env: []string{vars}}
	loc.YAML = append(loc.YAML, two)
	if err := cmdconfig.LoadFiles(context.Background(), &cfg, loc); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "named" || cfg.Other != 2 {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadUnresolvedEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "one.yaml", "name: a${LOAD_TEST_UNDEFINED}b\n")
	var ff cmdconfig.FilesFlag
	ff.SetDefaults("one.yaml")
	var cfg struct {
		Name string `yaml:"name"`
	}
	spec := cmdconfig.Spec{Path: cmdconfig.SearchPath("loadtest")}
	if _, err := cmdconfig.Load(context.Background(), &cfg, &ff, spec); err != nil || cfg.Name != "a${LOAD_TEST_UNDEFINED}b" {
		t.Fatalf("by default an unresolved reference is preserved: got %q, %v", cfg.Name, err)
	}
	if _, err := cmdconfig.Load(context.Background(), &cfg, &ff, spec, cmdyaml.WithUnresolvedEmpty()); err != nil || cfg.Name != "ab" {
		t.Fatalf("got %q, %v", cfg.Name, err)
	}
}

func TestLocatedExpandMapping(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOAD_TEST_FROM_ENV", "env")
	one := writeFile(t, dir, "one.yaml", "name: ${LOAD_TEST_FROM_ENV} ${LOAD_TEST_FROM_FILE} ${LOAD_TEST_DEFAULTED} ${LOAD_TEST_UNDEFINED}\n")
	vars := writeFile(t, dir, "vars.env", "LOAD_TEST_FROM_FILE=file\nLOAD_TEST_DEFAULTED=file-not-default\n")
	loc := &cmdconfig.Located{YAML: []string{one}, Env: []string{vars}}

	var unresolved envfile.Unresolved
	mapping, err := loc.ExpandMapping(
		envfile.DefaultsMapping(map[string]string{"LOAD_TEST_DEFAULTED": "default", "LOAD_TEST_UNDEFINED": ""}),
		unresolved.Mapping)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Name string `yaml:"name"`
	}
	if err := cmdconfig.LoadFiles(context.Background(), &cfg, loc, cmdyaml.WithUnresolvedEmpty(), cmdyaml.WithExpandMapping(mapping)); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Name, "env file file-not-default"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := unresolved.Names(), []string{"LOAD_TEST_UNDEFINED"}; !slices.Equal(got, want) {
		t.Errorf("unresolved: got %v, want %v", got, want)
	}

	loc.Env = append(loc.Env, filepath.Join(dir, "missing.env"))
	if _, err := loc.ExpandMapping(); err == nil || !strings.Contains(err.Error(), "missing.env") {
		t.Errorf("unexpected error: %v", err)
	}
}

type defaulted struct {
	Name  string `yaml:"name"`
	Extra string `yaml:"extra"`
}

func (d *defaulted) SetDefaults() {
	if d.Name == "" {
		d.Name = "default-name"
	}
	d.Extra = d.Name + "-extra"
}

func TestLoadSetDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, dir, "one.yaml", "name: from-file\n")
	spec := cmdconfig.Spec{Path: cmdconfig.SearchPath("loadtest")}

	var withFile defaulted
	var ff cmdconfig.FilesFlag
	ff.SetDefaults("one.yaml")
	if _, err := cmdconfig.Load(context.Background(), &withFile, &ff, spec); err != nil {
		t.Fatal(err)
	}
	if withFile.Name != "from-file" || withFile.Extra != "from-file-extra" {
		t.Errorf("got %+v", withFile)
	}

	// Defaults are applied when there are no files too.
	var none defaulted
	var empty cmdconfig.FilesFlag
	if _, err := cmdconfig.Load(context.Background(), &none, &empty, spec); err != nil {
		t.Fatal(err)
	}
	if none.Name != "default-name" || none.Extra != "default-name-extra" {
		t.Errorf("got %+v", none)
	}

	// ... but not when a file cannot be parsed.
	writeFile(t, dir, "bad.yaml", "unknown_field: 1\n")
	var bad defaulted
	var ffBad cmdconfig.FilesFlag
	if err := ffBad.Set("bad.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := cmdconfig.Load(context.Background(), &bad, &ffBad, spec); err == nil || bad.Name != "" {
		t.Errorf("got %+v, %v", bad, err)
	}
}
