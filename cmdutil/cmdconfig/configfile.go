// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// SearchPath returns the search path for configuration files for the
// application directory appDir, in the form expected by filepath.SplitList
// and hence by LocateFilePath. It contains, in order:
//
//  1. the current directory
//  2. appDir within the user's configuration directory as reported by
//     os.UserConfigDir, ie. $HOME/Library/Application Support/appDir on
//     macOS, $XDG_CONFIG_HOME/appDir or $HOME/.config/appDir elsewhere on
//     unix, and %AppData%\appDir on windows
//  3. on unix, including macOS, $HOME/.config/appDir
//
// Every directory in the path belongs to the user: no machine-wide location,
// such as /etc/appDir, is searched. Duplicates are omitted, so on a unix
// system where os.UserConfigDir is itself $HOME/.config the directory
// appears once rather than twice.
//
// A directory is included whether or not it exists; locating a file in one
// that does not is simply a miss.
//
// See SearchPathUserDirs for a path that leaves the current directory out.
func SearchPath(appDir string) string {
	return SearchPathFor(appDir, appDir)
}

// SearchPathFor is like SearchPath for an application that uses a different
// directory name in the user's configuration directory, userDir, than in
// $HOME/.config, xdgDir.
func SearchPathFor(userDir, xdgDir string) string {
	return joinSearchPath(append([]string{"."}, userConfigDirs(userDir, xdgDir)...))
}

// SearchPathUserDirs is SearchPath without the current directory: it contains
// the directories that belong to the user, ie. items 2 and 3 of the list in
// SearchPath, and nothing else.
//
// It is for a command that is run in arbitrary directories, where a
// configuration file in the current one is more likely to belong to whatever
// is being worked on, or to another application, than to the command itself;
// reading it means at best ignoring its contents and at worst failing to parse
// them. Such a command can still be pointed at a file in the current directory
// by being given its name, which it resolves against SearchPath instead.
//
// The result is empty when the user has no configuration directory at all,
// which takes os.UserConfigDir failing and, on unix, $HOME being unset. Both
// Spec and LocateFilePath read an empty path as the current directory, so a
// command that must not fall back to it has to say so itself.
func SearchPathUserDirs(appDir string) string {
	return joinSearchPath(userConfigDirs(appDir, appDir))
}

// userConfigDirs returns the configuration directories that belong to the
// user, ie. a search path without the current directory in it, for an
// application named userDir in the user's configuration directory and xdgDir
// in $HOME/.config.
func userConfigDirs(userDir, xdgDir string) []string {
	var dirs []string
	if configDir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(configDir, userDir))
	}
	return withXDGConfigDir(dirs, xdgDir)
}

// joinSearchPath returns dirs in the form expected by filepath.SplitList,
// with any directory that has already appeared removed.
func joinSearchPath(dirs []string) string {
	return strings.Join(dedup(dirs), string(os.PathListSeparator))
}

// dedup returns dirs with any directory that has already appeared removed,
// retaining the order of those that remain.
func dedup(dirs []string) []string {
	seen := make(map[string]bool, len(dirs))
	deduped := dirs[:0:0]
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		deduped = append(deduped, dir)
	}
	return deduped
}

// LocateFilePath searches for the first configuration file that matches
// the given name in path (as expected by filepath.SplitList). The suffixes are
// the filename suffixes to try, in order of preference, and include any
// leading '.', eg. ".yaml", ".yml". The search order is as follows:
//  1. If name is an absolute path and the file exists, it is returned, if it
//     does not exist an empty string is returned. If it exists and an error
//     other than "file not found" occurs, the filename is returned so that the
//     caller can attempt to open it and handle the error.
//  2. For each directory in path, the following are checked in order:
//     a. dir/name<suffix> for each suffix
//     b. dir/.name<suffix> for each suffix
//
// 3. If no matching file is found, an empty string is returned.
//
// If name already ends in one of the suffixes then only that suffix is used,
// so that dir/name.yaml is never tested for as dir/name.yaml.yml. If no
// suffixes are supplied then name is used as is, ie. only dir/name and
// dir/.name are tested for.
//
// The suffixes play no part when name is an absolute path: such a name is
// used exactly as given, so /dir/name is never tested for as /dir/name.yaml.
func LocateFilePath(name, path string, suffixes ...string) string {
	if filepath.IsAbs(name) {
		return locateAbsConfigFile(name)
	}
	base, suffixes := baseAndSuffixes(name, suffixes)
	for _, dir := range filepath.SplitList(path) {
		for _, prefix := range []string{"", "."} {
			for _, suffix := range suffixes {
				file := filepath.Join(dir, prefix+base+suffix)
				if _, err := os.Stat(file); err == nil {
					return file
				}
			}
		}
	}
	return ""
}

func locateAbsConfigFile(name string) string {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		// Return the path so callers get the underlying error when they open/read it.
		return name
	}
	if fi.IsDir() {
		return ""
	}
	return name
}

// baseAndSuffixes determines the name and the suffixes to append to it.
// Ensure that we don't test for x.yaml.yml or vice versa by stripping any
// suffix that name already ends in and only testing for that one suffix.
// With no suffixes to choose from the name is used as is.
func baseAndSuffixes(name string, suffixes []string) (string, []string) {
	if len(suffixes) == 0 {
		return name, []string{""}
	}
	for _, suffix := range suffixes {
		if len(suffix) > 0 && strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix), []string{suffix}
		}
	}
	return name, suffixes
}

// LocateFile searches for a configuration file with the given name
// using the search path returned by SearchPath, ie. it is
// LocateFilePath applied to that path. If name is an absolute path it
// is used directly. If no matching file is found an empty string is
// returned.
func LocateFile(name, appDir string, suffixes ...string) string {
	return LocateFilePath(name, SearchPath(appDir), suffixes...)
}

// LocateFiles searches for each of the named configuration files
// using the search path returned by SearchPath, resolving each name
// exactly as LocateFile resolves it. It returns the files that
// were found, in the order their names were supplied, and the names that
// were not found.
func LocateFiles(appDir string, suffixes []string, names ...string) (files, missing []string) {
	return LocateFilesPath(SearchPath(appDir), suffixes, names...)
}

// LocateFilesPath searches for each of the named configuration files
// using the specified search path and suffixes. It returns the files that
// were found, in the order their names were supplied, and the names that
// were not found.
func LocateFilesPath(path string, suffixes []string, names ...string) (files, missing []string) {
	for _, name := range names {
		if file := LocateFilePath(name, path, suffixes...); len(file) > 0 {
			files = append(files, file)
		} else {
			missing = append(missing, name)
		}
	}
	return files, missing
}

type fileSpec struct {
	defaults []string
	values   []string
}

func (s *fileSpec) usable(ignoreDefaults bool) []string {
	if ignoreDefaults {
		return s.values
	}
	if len(s.values) == 0 {
		return s.defaults
	}
	return s.values
}

func (s *fileSpec) locatePath(path string, ignoreDefault bool, suffixes []string) ([]string, []string) {
	return LocateFilesPath(path, suffixes, s.usable(ignoreDefault)...)
}

// FileType represents the type of configuration file: YAML, .env, or JSON.
type FileType int

const (
	FileTypeYAML FileType = iota
	FileTypeEnv
	FileTypeJSON
)

var fileTypeSuffixes = map[FileType][]string{
	FileTypeYAML: {".yaml", ".yml"},
	FileTypeEnv:  {".env"},
	FileTypeJSON: {".json"},
}

// String implements fmt.Stringer.
func (t FileType) String() string {
	switch t {
	case FileTypeYAML:
		return "YAML"
	case FileTypeEnv:
		return "environment variables"
	case FileTypeJSON:
		return "JSON"
	}
	return fmt.Sprintf("FileType(%d)", int(t))
}

// typeOf returns the type of file that name is, from its suffix.
func typeOf(name string) (FileType, bool) {
	for typ, suffixes := range fileTypeSuffixes {
		for _, suffix := range suffixes {
			if strings.HasSuffix(name, suffix) {
				return typ, true
			}
		}
	}
	return 0, false
}

// FilesFlag represents a command-line flag that can hold multiple YAML, json
// and .env files. It accepts comma-separated values and can also be repeated.
// The type of each file is determined by its suffix. Files that are set,
// on the command line, replace the defaults, see SetDefaults, and are
// located using Locate.
type FilesFlag struct {
	yamlFiles fileSpec
	envFiles  fileSpec
	jsonFiles fileSpec
}

func (f *FilesFlag) spec(typ FileType) *fileSpec {
	switch typ {
	case FileTypeYAML:
		return &f.yamlFiles
	case FileTypeEnv:
		return &f.envFiles
	case FileTypeJSON:
		return &f.jsonFiles
	}
	return nil
}

// Set parses a comma-separated list of file paths and categorizes them
// into YAML, .env, and JSON files based on their suffixes. The flag
// may be repeated on the command line.
func (f *FilesFlag) Set(vals string) error {
	for v := range strings.SplitSeq(vals, ",") {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		typ, ok := typeOf(v)
		if !ok {
			return fmt.Errorf("unsupported file type: %s", v)
		}
		s := f.spec(typ)
		s.values = append(s.values, v)
	}
	return nil
}

// SetDefault replaces the default files with the comma-separated list of
// file paths, categorizing them as Set does. It is called by
// flags.RegisterFlagsInStruct to set the default from the struct tag or from
// DefaultValues, and so is idempotent.
func (f *FilesFlag) SetDefault(vals string) error {
	var defaults FilesFlag
	for v := range strings.SplitSeq(vals, ",") {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		typ, ok := typeOf(v)
		if !ok {
			return fmt.Errorf("unsupported file type: %s", v)
		}
		s := defaults.spec(typ)
		s.values = append(s.values, v)
	}
	f.yamlFiles.defaults = defaults.yamlFiles.values
	f.envFiles.defaults = defaults.envFiles.values
	f.jsonFiles.defaults = defaults.jsonFiles.values
	return nil
}

// SetDefaults adds comma-separated lists of file paths, categorized as Set
// does, to the default files. Files with an unsupported suffix are ignored.
func (f *FilesFlag) SetDefaults(files ...string) {
	for _, file := range files {
		for v := range strings.SplitSeq(file, ",") {
			if typ, ok := typeOf(strings.TrimSpace(v)); ok {
				s := f.spec(typ)
				s.defaults = append(s.defaults, strings.TrimSpace(v))
			}
		}
	}
}

// DefaultValues returns a comma-separated string of all default file paths
// for YAML, .env, and JSON files.
func (f *FilesFlag) DefaultValues() string {
	return strings.Join(slices.Concat(f.yamlFiles.defaults, f.envFiles.defaults, f.jsonFiles.defaults), ",")
}

// String implements flag.Value, and returns the files that were set.
func (f *FilesFlag) String() string {
	return strings.Join(slices.Concat(f.yamlFiles.values, f.envFiles.values, f.jsonFiles.values), ",")
}

// Files returns a copy of the list of files of the specified type set in the
// flag or the default files if none were set. Use HasSet to check if any were
// explicitly set.
func (f *FilesFlag) Files(typ FileType, ignoreDefaults bool) []string {
	s := f.spec(typ)
	if s == nil {
		return nil
	}
	return slices.Clone(s.usable(ignoreDefaults))
}

// HasSet returns true if any files of the specified type were explicitly set
// in the flag.
func (f *FilesFlag) HasSet(typ FileType) bool {
	s := f.spec(typ)
	return s != nil && len(s.values) > 0
}

// LocateFilesPath locates the files of the specified type, in path, returning
// those that were found and the names of those that were not. The files are
// those set in the flag, or the defaults if none were set unless ignoreDefault
// is true.
func (f *FilesFlag) LocateFilesPath(typ FileType, ignoreDefault bool, path string) ([]string, []string) {
	s := f.spec(typ)
	if s == nil {
		return nil, nil
	}
	return s.locatePath(path, ignoreDefault, fileTypeSuffixes[typ])
}

// Spec specifies how the files of a FilesFlag are located.
type Spec struct {
	// Path is the search path for the files, in the form expected by
	// filepath.SplitList, see SearchPath and SearchPathFor. It defaults to the
	// current directory.
	Path string
	// EnvVars names, per type of file, an environment variable whose value is
	// a comma-separated list of files, which are used in place of the defaults
	// if no files of that type were set in the flag. Files named by the
	// environment variable are treated as having been set.
	EnvVars map[FileType]string
	// FirstOnly lists the types of file for which at most one file is
	// wanted: the first of those named that is found. Otherwise all of
	// those named that are found are returned, in the order named.
	FirstOnly []FileType
}

// Located is the result of locating the files of a FilesFlag.
type Located struct {
	YAML, Env, JSON []string
	// SearchPath is the path that the files were searched for in.
	SearchPath string
}

// Files returns the files of the specified type.
func (l *Located) Files(typ FileType) []string {
	switch typ {
	case FileTypeYAML:
		return l.YAML
	case FileTypeEnv:
		return l.Env
	case FileTypeJSON:
		return l.JSON
	}
	return nil
}

// Locate locates the files of each type held by the flag, as per spec. The
// files of a type are, in order of precedence, those set in the flag, those
// named by the environment variable for that type, and the defaults. Files
// that are set, in the flag or by the environment variable, must exist and
// an error is returned for any that does not. Defaults are optional: the
// ones that do not exist are omitted, and it is for the caller to decide
// whether it is an error for there to be none.
func (f *FilesFlag) Locate(spec Spec) (*Located, error) {
	path := spec.Path
	if path == "" {
		path = "."
	}
	loc := &Located{SearchPath: path}
	for _, typ := range []FileType{FileTypeYAML, FileTypeEnv, FileTypeJSON} {
		names, specified := f.names(typ, spec)
		files, missing := LocateFilesPath(path, fileTypeSuffixes[typ], names...)
		first := slices.Contains(spec.FirstOnly, typ)
		switch {
		case first && len(files) > 0:
			files = files[:1]
		case first && specified && len(names) > 0:
			return nil, notFound(typ, names, path)
		case !first && specified && len(missing) > 0:
			return nil, notFound(typ, missing, path)
		}
		switch typ {
		case FileTypeYAML:
			loc.YAML = files
		case FileTypeEnv:
			loc.Env = files
		case FileTypeJSON:
			loc.JSON = files
		}
	}
	return loc, nil
}

// names returns the names of the files of the specified type to be located
// and whether they were specified, ie. are not defaults.
func (f *FilesFlag) names(typ FileType, spec Spec) (names []string, specified bool) {
	s := f.spec(typ)
	if len(s.values) > 0 {
		return s.values, true
	}
	if env := spec.EnvVars[typ]; env != "" {
		for v := range strings.SplitSeq(os.Getenv(env), ",") {
			if v = strings.TrimSpace(v); v != "" {
				names = append(names, v)
			}
		}
		if len(names) > 0 {
			return names, true
		}
	}
	return s.defaults, false
}

func notFound(typ FileType, names []string, path string) error {
	return fmt.Errorf("%v file(s) %v not found in any of %v", typ, strings.Join(names, ", "), strings.Join(filepath.SplitList(path), ", "))
}
