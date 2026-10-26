// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdconfig"
)

// yamlSuffixes are the suffixes of YAML files and are used here as a
// representative set.
var yamlSuffixes = []string{".yaml", ".yml"}

func writeConfigFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("key: value\n"), 0600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func testLocateConfigFilePathAbs(t *testing.T) {
	dir := t.TempDir()
	existing := writeConfigFile(t, dir, "config.yaml")

	// An absolute name is used as is: no suffix is appended to it and path is
	// not consulted.
	if got := cmdconfig.LocateFilePath(existing, "", yamlSuffixes...); got != existing {
		t.Errorf("got %q, want %q", got, existing)
	}
	if got := cmdconfig.LocateFilePath(existing, dir); got != existing {
		t.Errorf("got %q, want %q", got, existing)
	}
	for _, name := range []string{
		filepath.Join(dir, "missing.yaml"), // does not exist
		filepath.Join(dir, "config"),       // exists only with a suffix
		dir,                                // a directory, not a file
	} {
		if got := cmdconfig.LocateFilePath(name, dir, yamlSuffixes...); got != "" {
			t.Errorf("%v: got %q, want an empty string", name, got)
		}
	}
}

func testLocateConfigFilePathSuffixes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		createFiles []string
		suffixes    []string
		locate      string
		want        string
	}{
		{
			"the suffixes are tried in the order supplied",
			[]string{"app.jsonc", "app.json"},
			[]string{".json", ".jsonc"},
			"app",
			"app.json",
		},
		{
			"reversing the suffixes reverses the preference",
			[]string{"app.jsonc", "app.json"},
			[]string{".jsonc", ".json"},
			"app",
			"app.jsonc",
		},
		{
			// The undotted name outranks the dot prefixed one whichever way
			// around the suffixes are.
			"an undotted name outranks a dot prefixed one",
			[]string{".app.json", "app.jsonc"},
			[]string{".json", ".jsonc"},
			"app",
			"app.jsonc",
		},
		{
			"a dot prefixed name is found when there is no undotted one",
			[]string{".app.jsonc"},
			[]string{".json", ".jsonc"},
			"app",
			".app.jsonc",
		},
		{
			// A suffix need not be an extension.
			"suffixes need not be extensions",
			[]string{"app-config.json"},
			[]string{"-config.json"},
			"app",
			"app-config.json",
		},
		{
			"a name that already ends in a suffix does not acquire a second one",
			[]string{"app.json", "app.json.jsonc"},
			[]string{".jsonc", ".json"},
			"app.json",
			"app.json",
		},
		{
			// Only the suffix the name ends in is used, so the file that
			// would have been preferred for a bare name is not considered.
			"a name that ends in a suffix pins that suffix",
			[]string{"app.jsonc", "app.json"},
			[]string{".jsonc", ".json"},
			"app.json",
			"app.json",
		},
		{
			"with no suffixes the name is used as is",
			[]string{"app"},
			nil,
			"app",
			"app",
		},
		{
			"with no suffixes a dot prefixed name is still found",
			[]string{".app"},
			nil,
			"app",
			".app",
		},
		{
			"with no suffixes a suffixed file is not found",
			[]string{"app.json"},
			nil,
			"app",
			"",
		},
		{
			"an empty suffix is usable alongside others",
			[]string{"app"},
			[]string{".json", ""},
			"app",
			"app",
		},
		{
			"nothing matches",
			[]string{"other.json"},
			[]string{".json"},
			"app",
			"",
		},
	} {
		dir := t.TempDir()
		for _, name := range tc.createFiles {
			writeConfigFile(t, dir, name)
		}
		want := tc.want
		if want != "" {
			want = filepath.Join(dir, want)
		}
		if got := cmdconfig.LocateFilePath(tc.locate, dir, tc.suffixes...); got != want {
			t.Errorf("%v: got %q, want %q", tc.name, got, want)
		}
	}
}

func testLocateConfigFilePathSearchPath(t *testing.T) {
	dir1, dir2, dir3 := t.TempDir(), t.TempDir(), t.TempDir()
	path := strings.Join([]string{dir1, dir2, dir3}, string(filepath.ListSeparator))

	// Earlier directories in the path win, even over a more preferred suffix
	// or an undotted name in a later one.
	writeConfigFile(t, dir2, "service.json")
	want := filepath.Join(dir2, "service.json")
	if got := cmdconfig.LocateFilePath("service", path, ".json", ".jsonc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	writeConfigFile(t, dir1, ".service.jsonc")
	want = filepath.Join(dir1, ".service.jsonc")
	if got := cmdconfig.LocateFilePath("service", path, ".json", ".jsonc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	writeConfigFile(t, dir1, "service.jsonc")
	want = filepath.Join(dir1, "service.jsonc")
	if got := cmdconfig.LocateFilePath("service", path, ".json", ".jsonc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testLocateConfigFilePathNotFound(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ name, path string }{
		{"nonexistent", dir},
		{"nonexistent", ""},
		{"", dir},
	} {
		if got := cmdconfig.LocateFilePath(tc.name, tc.path, yamlSuffixes...); got != "" {
			t.Errorf("(%q, %q): got %q, want an empty string", tc.name, tc.path, got)
		}
	}
}

func TestLocateConfigFilePath(t *testing.T) {
	t.Run("absolute name", testLocateConfigFilePathAbs)
	t.Run("suffix handling", testLocateConfigFilePathSuffixes)
	t.Run("search path ordering", testLocateConfigFilePathSearchPath)
	t.Run("not found", testLocateConfigFilePathNotFound)
}

// configDirs holds the directories that SearchPath includes, in search
// order, for the appDir passed to redirectConfigDir.
type configDirs struct {
	// cwd is the working directory, which SearchPath includes as ".", so
	// matches found there are relative.
	cwd string
	// user is os.UserConfigDir()/appDir, ie.
	// $HOME/Library/Application Support/appDir on macOS.
	user string
	// xdg is $HOME/.config/appDir, the XDG Base Directory default, on unix
	// (darwin included) and is empty on platforms where no such directory is
	// included.
	xdg string
}

// redirectConfigDir points every directory that SearchPath includes at a
// temporary one: the working directory, the user configuration directory (via
// whichever environment variable os.UserConfigDir consults on this platform)
// and, on unix, $HOME/.config. On unix XDG_CONFIG_HOME is pointed at a
// directory of its own so that the user configuration directory is distinct
// from $HOME/.config even where os.UserConfigDir would otherwise return the
// latter; macOS ignores XDG_CONFIG_HOME. The returned directories have all
// been created.
func redirectConfigDir(t *testing.T, appDir string) configDirs {
	t.Helper()
	dirs := configDirs{cwd: t.TempDir()}
	home := t.TempDir()
	t.Chdir(dirs.cwd)
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", home)
	case "plan9":
		t.Setenv("home", home)
	default: // darwin, ios and unix
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
		dirs.xdg = filepath.Join(home, ".config", appDir)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir: %v", err)
	}
	if !strings.HasPrefix(configDir, home) {
		t.Skipf("os.UserConfigDir returned %v, which this test cannot redirect on %v", configDir, runtime.GOOS)
	}
	dirs.user = filepath.Join(configDir, appDir)
	for _, dir := range []string{dirs.user, dirs.xdg} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return dirs
}

func testConfigFilePathEntries(t *testing.T) {
	dirs := redirectConfigDir(t, "app")

	// The working directory comes first, as ".", then the user configuration
	// directory, then the XDG configuration directory where there is one,
	// each with appDir appended to it.
	want := []string{".", dirs.user}
	if dirs.xdg != "" {
		want = append(want, dirs.xdg)
	}
	got := filepath.SplitList(cmdconfig.SearchPath("app"))
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// appDir is appended literally, so a different application gets a
	// different search path.
	if other := filepath.SplitList(cmdconfig.SearchPath("other")); slices.Equal(other, got) {
		t.Errorf("the search paths for %q and %q are identical: %v", "app", "other", got)
	}
}

// testConfigFilePathNoDuplicates verifies that a directory appears in the
// search path once, however many of the sources it is reported by: on a unix
// system that is not macOS, os.UserConfigDir is $HOME/.config when
// XDG_CONFIG_HOME is unset, which is the directory appended for XDG as well.
func testConfigFilePathNoDuplicates(t *testing.T) {
	home := t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	dirs := filepath.SplitList(cmdconfig.SearchPath("app"))
	seen := map[string]bool{}
	for _, dir := range dirs {
		if seen[dir] {
			t.Errorf("%v appears more than once in %v", dir, dirs)
		}
		seen[dir] = true
	}

	// The XDG directory is still searched, whether it was contributed once or
	// twice.
	if runtime.GOOS != "windows" && runtime.GOOS != "plan9" {
		xdg := filepath.Join(home, ".config", "app")
		if !slices.Contains(dirs, xdg) {
			t.Errorf("%v is not in %v", xdg, dirs)
		}
	}
}

func testConfigFilePathNoUserConfigDir(t *testing.T) {
	t.Chdir(t.TempDir())
	// Leave os.UserConfigDir with nothing to work from on any platform: it
	// consults XDG_CONFIG_HOME and HOME on unix, HOME on darwin, AppData on
	// windows and home on plan9.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("AppData", "")
	t.Setenv("home", "")
	if dir, err := os.UserConfigDir(); err == nil {
		t.Skipf("os.UserConfigDir still resolves to %v on %v", dir, runtime.GOOS)
	}
	// The error is not surfaced; the search path falls back to the working
	// directory alone.
	if got, want := cmdconfig.SearchPath("app"), "."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testConfigFilePathLocates(t *testing.T) {
	dirs := redirectConfigDir(t, "app")
	want := writeConfigFile(t, dirs.user, "cfg.yaml")

	// The returned path is intended for use with LocateFilePath.
	path := cmdconfig.SearchPath("app")
	if got := cmdconfig.LocateFilePath("cfg", path, yamlSuffixes...); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The working directory is searched first, and matches there are
	// relative.
	writeConfigFile(t, dirs.cwd, "cfg.yml")
	if got, want := cmdconfig.LocateFilePath("cfg", path, yamlSuffixes...), "cfg.yml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestConfigFilePath covers the search path that LocateFilePath is
// typically used with. Note that os.UserConfigDir is
// $HOME/Library/Application Support on macOS, not $HOME/.config, so on that
// platform both appear in the path.
func TestConfigFilePath(t *testing.T) {
	t.Run("entries and ordering", testConfigFilePathEntries)
	t.Run("no duplicate directories", testConfigFilePathNoDuplicates)
	t.Run("no user configuration directory", testConfigFilePathNoUserConfigDir)
	t.Run("locates a file", testConfigFilePathLocates)
}

func testLocateConfigFileInDir(t *testing.T) {
	dirs := redirectConfigDir(t, "app")

	// Nothing to find.
	if got := cmdconfig.LocateFile("cfg", "app", yamlSuffixes...); len(got) != 0 {
		t.Errorf("got %q, want no file", got)
	}

	// The user configuration directory is searched, and a match there is
	// returned as an absolute path.
	want := writeConfigFile(t, dirs.user, "cfg.yaml")
	if got := cmdconfig.LocateFile("cfg", "app", yamlSuffixes...); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The working directory is searched first, and a match there is relative.
	writeConfigFile(t, dirs.cwd, "cfg.yml")
	if got, want := cmdconfig.LocateFile("cfg", "app", yamlSuffixes...), "cfg.yml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The suffixes are the caller's to choose, and with none the name is used
	// as it is.
	writeConfigFile(t, dirs.cwd, "vars.env")
	if got, want := cmdconfig.LocateFile("vars", "app", ".env"), "vars.env"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := cmdconfig.LocateFile("vars", "app"); len(got) != 0 {
		t.Errorf("got %q, want no file: vars has no suffix to match", got)
	}
	if got, want := cmdconfig.LocateFile("vars.env", "app"), "vars.env"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// An absolute name is used directly.
	if got := cmdconfig.LocateFile(want, "app", yamlSuffixes...); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testLocateConfigFilesInDir(t *testing.T) {
	dirs := redirectConfigDir(t, "app")
	writeConfigFile(t, dirs.cwd, "one.yaml")
	writeConfigFile(t, dirs.user, "two.yml")

	// The files are returned in the order their names were supplied, and the
	// names that were not found are reported separately.
	files, missing := cmdconfig.LocateFiles("app", yamlSuffixes, "two", "nope", "one")
	if got, want := files, []string{filepath.Join(dirs.user, "two.yml"), "one.yaml"}; !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if got, want := missing, []string{"nope"}; !slices.Equal(got, want) {
		t.Errorf("missing = %v, want %v", got, want)
	}

	// A repeated name is resolved each time it appears.
	files, missing = cmdconfig.LocateFiles("app", yamlSuffixes, "one", "one", "nope", "nope")
	if got, want := files, []string{"one.yaml", "one.yaml"}; !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if got, want := missing, []string{"nope", "nope"}; !slices.Equal(got, want) {
		t.Errorf("missing = %v, want %v", got, want)
	}

	// No names at all yields nothing, as does a name with no suffix to match.
	files, missing = cmdconfig.LocateFiles("app", yamlSuffixes)
	if len(files) != 0 || len(missing) != 0 {
		t.Errorf("files = %v, missing = %v, want both empty", files, missing)
	}
	files, missing = cmdconfig.LocateFiles("app", nil, "one")
	if got, want := missing, []string{"one"}; len(files) != 0 || !slices.Equal(got, want) {
		t.Errorf("files = %v, missing = %v, want no files and %v", files, got, want)
	}
}

// TestLocateConfigFileInDir covers the convenience wrappers that apply
// LocateFilePath to the search path returned by SearchPath, and which
// cmdyaml's wrappers delegate to.
func TestLocateConfigFileInDir(t *testing.T) {
	t.Run("single name", testLocateConfigFileInDir)
	t.Run("several names", testLocateConfigFilesInDir)
}
