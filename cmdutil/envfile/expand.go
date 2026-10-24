// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile

import (
	"os"
	"strings"
)

// maxExpansions bounds the depth to which a value is expanded, so that a
// definition that refers to itself, directly or through others, terminates.
const maxExpansions = 10

// ExpandingMapping returns a mapping, with the signature expected by os.Expand,
// that takes its values from the process environment and, for any variable
// that the environment does not define, from vars. The shell environment
// overrides the definitions, so that a value configured for everyday use can
// be changed for a single invocation without editing the file. Note that a
// variable set to an empty value is indistinguishable from one that is not
// set, so an empty value in the environment does not hide a definition in
// vars.
//
// A value taken from vars may itself contain references, as in
//
//	PROFILE=$HOME/Downloads/example.provisionprofile
//
// and those are expanded too, from the environment first and then from vars,
// since Parse does not expand them when it reads a file and a single pass over
// a configuration file would otherwise leave the $HOME above in the value. A
// reference that resolves to nothing is left as ${NAME}, so that it is
// visible rather than silently empty, and expansion is bounded so that a
// definition referring to itself terminates.
//
// Values taken from the environment are used as they are: the shell has
// already expanded them, and a value that merely contains a '$', such as a
// password, must not be altered.
//
// A nil vars yields the process environment alone.
func ExpandingMapping(vars *Variables) func(string) string {
	var lookup func(name string, depth int) string
	lookup = func(name string, depth int) string {
		if value, ok := os.LookupEnv(name); ok && len(value) > 0 {
			return value
		}
		value, ok := vars.Lookup(name)
		if !ok || depth >= maxExpansions || !strings.Contains(value, "$") {
			return value
		}
		return os.Expand(value, func(ref string) string {
			if expanded := lookup(ref, depth+1); len(expanded) > 0 {
				return expanded
			}
			return "${" + ref + "}"
		})
	}
	return func(name string) string {
		return lookup(name, 0)
	}
}
