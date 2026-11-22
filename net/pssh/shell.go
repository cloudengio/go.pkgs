// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import "strings"

// ShellQuote quotes s so that a POSIX shell treats it as a single literal
// word, with no expansion of any kind.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShellJoin quotes each of args with ShellQuote and joins them with spaces,
// to form a command line that runs args[0] with the remaining arguments.
func ShellJoin(args ...string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = ShellQuote(a)
	}
	return strings.Join(quoted, " ")
}

// loginShellCommand returns a command that runs cmd in the user's login shell,
// falling back to sh if the server does not set SHELL.
func loginShellCommand(cmd string) string {
	return `exec "${SHELL:-/bin/sh}" -l -c ` + ShellQuote(cmd)
}
