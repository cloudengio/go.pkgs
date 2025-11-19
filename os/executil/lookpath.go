// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import "strings"

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
	prefix := key + "="
	for _, e := range env {
		if after, ok := strings.CutPrefix(e, prefix); ok {
			return after, true
		}
	}
	return "", false
}
