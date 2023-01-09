// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"io"
	"io/fs"
)

// WriteFS extends fs.FS to add a Create method.
type WriteFS interface {
	fs.FS
	Create(name string, mode fs.FileMode) (io.WriteCloser, string, error)
}
