// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig_test

import (
	"flag"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdconfig"
	"cloudeng.io/cmdutil/flags"
)

func TestFilesFlagSet(t *testing.T) {
	var f cmdconfig.FilesFlag
	for _, v := range []string{"a.yaml,b.yml", "c.env", "d.json, e.yaml,", "f.env"} {
		if err := f.Set(v); err != nil {
			t.Fatal(err)
		}
	}
	for typ, want := range map[cmdconfig.FileType][]string{
		cmdconfig.FileTypeYAML: {"a.yaml", "b.yml", "e.yaml"},
		cmdconfig.FileTypeEnv:  {"c.env", "f.env"},
		cmdconfig.FileTypeJSON: {"d.json"},
	} {
		if got := f.Files(typ, false); !slices.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", typ, got, want)
		}
		if !f.HasSet(typ) {
			t.Errorf("%v: not set", typ)
		}
	}
	if err := f.Set("x.txt"); err == nil || !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("unexpected error: %v", err)
	}
	// Returned slices are copies.
	f.Files(cmdconfig.FileTypeYAML, false)[0] = "changed"
	if got := f.Files(cmdconfig.FileTypeYAML, false)[0]; got != "a.yaml" {
		t.Errorf("got %v", got)
	}
}

func TestFilesFlagDefaults(t *testing.T) {
	var f cmdconfig.FilesFlag
	f.SetDefaults("a.yaml,b.env", "c.json", "ignored.txt")
	if got, want := f.DefaultValues(), "a.yaml,b.env,c.json"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// DefaultValues does not alias the defaults.
	_ = f.DefaultValues()
	f.SetDefaults("d.yaml")
	if got, want := f.Files(cmdconfig.FileTypeYAML, false), []string{"a.yaml", "d.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if f.HasSet(cmdconfig.FileTypeYAML) {
		t.Error("defaults are not set")
	}
	// Setting values replaces the defaults, unless they are ignored.
	if err := f.Set("x.yaml"); err != nil {
		t.Fatal(err)
	}
	if got, want := f.Files(cmdconfig.FileTypeYAML, false), []string{"x.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.Files(cmdconfig.FileTypeEnv, false), []string{"b.env"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := f.Files(cmdconfig.FileTypeEnv, true); len(got) != 0 {
		t.Errorf("got %v", got)
	}
	// SetDefault replaces the defaults, and so is idempotent.
	for range 2 {
		if err := f.SetDefault(f.DefaultValues()); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := f.DefaultValues(), "a.yaml,d.yaml,b.env,c.json"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := f.SetDefault("z.txt"); err == nil {
		t.Error("expected an error")
	}
}

// TestFilesFlagRegistered checks that the defaults are registered as defaults,
// and not as values that were set.
func TestFilesFlagRegistered(t *testing.T) {
	var s struct {
		Config cmdconfig.FilesFlag `flag:"config,,configuration files"`
		Other  cmdconfig.FilesFlag `flag:"other,'a.yaml,b.env',other files"`
	}
	s.Config.SetDefaults("c.yaml", "d.env")
	fs := &flag.FlagSet{}
	if err := flags.RegisterFlagsInStruct(fs, "flag", &s, nil, nil); err != nil {
		t.Fatal(err)
	}
	if s.Config.HasSet(cmdconfig.FileTypeYAML) || s.Other.HasSet(cmdconfig.FileTypeYAML) {
		t.Error("defaults are not values")
	}
	if got, want := s.Config.DefaultValues(), "c.yaml,d.env"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := s.Other.DefaultValues(), "a.yaml,b.env"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := fs.Parse([]string{"--config=x.yaml", "--config=y.yaml"}); err != nil {
		t.Fatal(err)
	}
	if got, want := s.Config.Files(cmdconfig.FileTypeYAML, false), []string{"x.yaml", "y.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFilesFlagLocateFilesPath(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, "one.yaml")
	writeConfigFile(t, dir, ".two.yaml")
	writeConfigFile(t, dir, "vars.env")
	var f cmdconfig.FilesFlag
	f.SetDefaults("one.yaml", "two.yaml", "three.yaml", "vars.env")
	files, missing := f.LocateFilesPath(cmdconfig.FileTypeYAML, false, dir)
	if got, want := files, []string{filepath.Join(dir, "one.yaml"), filepath.Join(dir, ".two.yaml")}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := missing, []string{"three.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	files, missing = f.LocateFilesPath(cmdconfig.FileTypeEnv, false, dir)
	if got, want := files, []string{filepath.Join(dir, "vars.env")}; !slices.Equal(got, want) || len(missing) != 0 {
		t.Errorf("got %v, %v, want %v", got, missing, want)
	}
	if files, _ := f.LocateFilesPath(cmdconfig.FileTypeYAML, true, dir); len(files) != 0 {
		t.Errorf("defaults should be ignored, got %v", files)
	}
	if files, _ := f.LocateFilesPath(cmdconfig.FileTypeJSON, false, dir); len(files) != 0 {
		t.Errorf("got %v", files)
	}
	// Files that were set are the ones located.
	if err := f.Set("three.yaml,one.yaml"); err != nil {
		t.Fatal(err)
	}
	files, missing = f.LocateFilesPath(cmdconfig.FileTypeYAML, false, dir)
	if got, want := files, []string{filepath.Join(dir, "one.yaml")}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := missing, []string{"three.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFilesFlagLocate(t *testing.T) {
	dirs := redirectConfigDir(t, "app")
	writeConfigFile(t, dirs.user, "shared.yaml")
	writeConfigFile(t, dirs.cwd, "app.yaml")
	vars := writeConfigFile(t, dirs.user, "vars.env")
	spec := cmdconfig.Spec{Path: cmdconfig.SearchPath("app")}

	// Defaults: all those that exist, in order; the missing are not an error.
	var f cmdconfig.FilesFlag
	f.SetDefaults("shared.yaml", "app.yaml", "missing.yaml", "vars.env", "missing.env")
	loc, err := f.Locate(spec)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loc.YAML, []string{filepath.Join(dirs.user, "shared.yaml"), "app.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := loc.Env, []string{vars}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(loc.JSON) != 0 || loc.SearchPath != cmdconfig.SearchPath("app") {
		t.Errorf("got %+v", loc)
	}
	if got := loc.Files(cmdconfig.FileTypeYAML); !slices.Equal(got, loc.YAML) {
		t.Errorf("got %v", got)
	}

	// FirstOnly: only the first of those that exist.
	spec.FirstOnly = []cmdconfig.FileType{cmdconfig.FileTypeYAML}
	loc, err = f.Locate(spec)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loc.YAML, []string{filepath.Join(dirs.user, "shared.yaml")}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestFilesFlagLocateSpecified(t *testing.T) {
	dirs := redirectConfigDir(t, "app")
	writeConfigFile(t, dirs.cwd, "app.yaml")
	spec := cmdconfig.Spec{Path: cmdconfig.SearchPath("app")}
	var f cmdconfig.FilesFlag
	// Files that are set must exist.
	if err := f.Set("app.yaml,nope.yaml"); err != nil {
		t.Fatal(err)
	}
	spec.FirstOnly = nil
	if _, err := f.Locate(spec); err == nil || !strings.Contains(err.Error(), "nope.yaml") || strings.Contains(err.Error(), "app.yaml,") {
		t.Errorf("unexpected error: %v", err)
	}
	// ... but with FirstOnly any one of them will do.
	spec.FirstOnly = []cmdconfig.FileType{cmdconfig.FileTypeYAML}
	if loc, err := f.Locate(spec); err != nil || !slices.Equal(loc.YAML, []string{"app.yaml"}) {
		t.Errorf("got %v, %v", loc, err)
	}
	// ... unless none exists.
	var g cmdconfig.FilesFlag
	if err := g.Set("nope.yaml,nada.yml"); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Locate(spec); err == nil {
		t.Error("expected an error")
	}
}

func TestFilesFlagLocateEnvVar(t *testing.T) {
	dirs := redirectConfigDir(t, "app")
	writeConfigFile(t, dirs.user, "default.yaml")
	writeConfigFile(t, dirs.user, "fromenv.yaml")
	writeConfigFile(t, dirs.user, "fromflag.yaml")
	spec := cmdconfig.Spec{
		Path:    cmdconfig.SearchPath("app"),
		EnvVars: map[cmdconfig.FileType]string{cmdconfig.FileTypeYAML: "CMDCONFIG_TEST_CONFIG"},
	}
	var f cmdconfig.FilesFlag
	f.SetDefaults("default.yaml")

	t.Setenv("CMDCONFIG_TEST_CONFIG", "")
	loc, err := f.Locate(spec)
	if err != nil || len(loc.YAML) != 1 || filepath.Base(loc.YAML[0]) != "default.yaml" {
		t.Fatalf("got %v, %v", loc, err)
	}

	// The environment variable replaces the defaults.
	t.Setenv("CMDCONFIG_TEST_CONFIG", "fromenv")
	loc, err = f.Locate(spec)
	if err != nil || len(loc.YAML) != 1 || filepath.Base(loc.YAML[0]) != "fromenv.yaml" {
		t.Fatalf("got %v, %v", loc, err)
	}

	// ... and what it names must exist.
	t.Setenv("CMDCONFIG_TEST_CONFIG", "missing")
	if _, err := f.Locate(spec); err == nil {
		t.Error("expected an error")
	}

	// The flag takes precedence over the environment variable.
	t.Setenv("CMDCONFIG_TEST_CONFIG", "missing")
	if err := f.Set("fromflag.yaml"); err != nil {
		t.Fatal(err)
	}
	loc, err = f.Locate(spec)
	if err != nil || len(loc.YAML) != 1 || filepath.Base(loc.YAML[0]) != "fromflag.yaml" {
		t.Fatalf("got %v, %v", loc, err)
	}
}

func TestFilesFlagLocatePath(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writeConfigFile(t, second, "cfg.yaml")
	var f cmdconfig.FilesFlag
	f.SetDefaults("cfg.yaml")
	spec := cmdconfig.Spec{Path: strings.Join([]string{first, second}, string(filepath.ListSeparator))}
	loc, err := f.Locate(spec)
	if err != nil || !slices.Equal(loc.YAML, []string{filepath.Join(second, "cfg.yaml")}) || loc.SearchPath != spec.Path {
		t.Errorf("got %+v, %v", loc, err)
	}
}

func TestSearchPathFor(t *testing.T) {
	dirs := redirectConfigDir(t, "user-name")
	if dirs.xdg == "" {
		t.Skip("no XDG directory on this platform")
	}
	xdg := filepath.Join(filepath.Dir(dirs.xdg), "xdg-name")
	want := []string{".", dirs.user, xdg}
	if got := filepath.SplitList(cmdconfig.SearchPathFor("user-name", "xdg-name")); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// With the same name for both it is SearchPath.
	if got, want := cmdconfig.SearchPathFor("app", "app"), cmdconfig.SearchPath("app"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFilesFlagLocateDefaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeConfigFile(t, dir, "cfg.yaml")
	var f cmdconfig.FilesFlag
	f.SetDefaults("cfg.yaml")
	loc, err := f.Locate(cmdconfig.Spec{})
	if err != nil || !slices.Equal(loc.YAML, []string{"cfg.yaml"}) || loc.SearchPath != "." {
		t.Errorf("got %+v, %v", loc, err)
	}
}
