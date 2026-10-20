// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ConfigFilePath returns the search path for configuration files for the
// application directory appDir, in the form expected by filepath.SplitList
// and hence by LocateConfigFilePath. It contains, in order:
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
func ConfigFilePath(appDir string) string {
	dirs := []string{"."}
	if configDir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(configDir, appDir))
	}
	dirs = withXDGConfigDir(dirs, appDir)
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

// LocateConfigFilePath searches for the first configuration file that matches
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
func LocateConfigFilePath(name, path string, suffixes ...string) string {
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

// LocateConfigFileInDir searches for a configuration file with the given name
// using the search path returned by ConfigFilePath, ie. it is
// LocateConfigFilePath applied to that path. If name is an absolute path it
// is used directly. If no matching file is found an empty string is
// returned.
func LocateConfigFileInDir(name, appDir string, suffixes ...string) string {
	return LocateConfigFilePath(name, ConfigFilePath(appDir), suffixes...)
}

// LocateConfigFilesInDir searches for each of the named configuration files
// using the search path returned by ConfigFilePath, resolving each name
// exactly as LocateConfigFileInDir resolves it. It returns the files that
// were found, in the order their names were supplied, and the names that
// were not found.
func LocateConfigFilesInDir(appDir string, suffixes []string, names ...string) (files, missing []string) {
	path := ConfigFilePath(appDir)
	for _, name := range names {
		if file := LocateConfigFilePath(name, path, suffixes...); len(file) > 0 {
			files = append(files, file)
		} else {
			missing = append(missing, name)
		}
	}
	return files, missing
}
