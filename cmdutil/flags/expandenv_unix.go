// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package flags

import (
	"os"
	"strings"
)

// ExpandEnv is like os.ExpandEnv but supports 'pseudo' environment
// variables that have OS specific handling as follows:
//
// $USERHOME is replaced by $HOME on unix-like sytems and $HOMEDRIVE:\\$HOMEPATH
// on windows.
// On windows, / are replaced with \.
func ExpandEnv(e string) string {
	e = strings.ReplaceAll(e, "$USERHOME", "$HOME")
	return os.ExpandEnv(e)
}
