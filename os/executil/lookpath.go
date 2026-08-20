// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"os"
	"slices"
	"strings"
)

// ReplaceEnvVar replaces the value of an environment variable in the provided slice.
// If the variable does not exist, it is added to the slice.
func ReplaceEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// Getenv retrieves the value of an environment variable from the provided slice.
func Getenv(env []string, key string) (string, bool) {
	for _, e := range env {
		if i := strings.IndexByte(e, '='); i > 0 && e[:i] == key {
			return e[i+1:], true
		}
	}
	return "", false
}

// AppendMissingPathComponents treats pathList as a path list, ie. a list of paths
// separated by os.PathListSeparator. It returns a copy of pathList with any of
// the specified paths that do not already exist in pathList appended to the
// end of the returned string.
func AppendMissingPathComponents(pathList string, paths ...string) string {
	pathComponents := strings.Split(pathList, string(os.PathListSeparator))
	for _, c := range paths {
		if !slices.Contains(pathComponents, c) {
			pathComponents = append(pathComponents, c)
		}
	}
	return strings.Join(pathComponents, string(os.PathListSeparator))
}
