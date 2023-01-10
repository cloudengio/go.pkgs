// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"io/fs"

	"cloudeng.io/file/download"
)

type ErrorDetail struct {
	download.Result
	Error error
}

type Errors struct {
	Request   download.Request
	Container fs.FS
	Errors    []ErrorDetail
}
