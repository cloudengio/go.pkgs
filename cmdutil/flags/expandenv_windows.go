// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package flags

import (
	"os"
	"strings"
)

// ExpandEnv is like os.ExpandEnv but supports 'pseudo' environment
// variables that have OS specific handling as follows:
//
// On UNIX systems $USERHOME is replaced by $HOME.
// On Windows $USERHOME and $HOME are replaced by and $HOMEDRIVE:\\$HOMEPATH
// On Windows /'s are replaced with \'s.
func ExpandEnv(e string) string {
	e = strings.ReplaceAll(e, "$HOME", `$HOMEDRIVE$HOMEPATH`)
	e = strings.ReplaceAll(e, "$USERHOME", `$HOMEDRIVE$HOMEPATH`)
	return strings.ReplaceAll(os.ExpandEnv(e), `/`, `\`)
}
