// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download

import (
	"io/fs"

	"cloudeng.io/file"
)

// SimpleRequest is a simple implementation of Request.
type SimpleRequest struct {
	RequestedBy string
	FS          file.FS
	Filenames   []string
	Mode        fs.FileMode
}

func (cr SimpleRequest) Requester() string {
	return cr.RequestedBy
}

func (cr SimpleRequest) Container() file.FS {
	return cr.FS
}

func (cr SimpleRequest) Names() []string {
	return cr.Filenames
}

func (cr SimpleRequest) FileMode() fs.FileMode {
	return cr.Mode
}
