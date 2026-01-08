// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"fmt"
	"strings"

	"cloudeng.io/file"
	"cloudeng.io/file/download"
)

type ErrorDetail struct {
	download.Result
}

func (e ErrorDetail) Error() string {
	return fmt.Sprintf("%v: %v", e.Result.Name, e.Err)
}

type Errors struct {
	Request   download.Request
	Container file.FS
	Errors    []ErrorDetail
}

func (e Errors) String() string {
	var out strings.Builder
	for _, detail := range e.Errors {
		fmt.Fprintf(&out, "%v: %v\n", detail.Name, detail.Err)
	}
	return out.String()
}

func (e Errors) Error() string {
	return e.String()
}
