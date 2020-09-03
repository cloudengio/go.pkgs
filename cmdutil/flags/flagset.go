// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"flag"
	"strings"
)

// Defaults returns the output of PrintDefaults() as a string.
func Defaults(fs *flag.FlagSet) string {
	out := &strings.Builder{}
	orig := fs.Output()
	defer fs.SetOutput(orig)
	fs.SetOutput(out)
	fs.PrintDefaults()
	return out.String()
}

// NamesAndDefault returns a string with flag names and their default
// values.
func NamesAndDefault(fs *flag.FlagSet) string {
	summary := []string{}
	fs.VisitAll(func(fl *flag.Flag) {
		summary = append(summary, "--"+fl.Name+"="+fl.DefValue)
	})
	return fs.Name() + " [" + strings.Join(summary, " ") + "]"
}
