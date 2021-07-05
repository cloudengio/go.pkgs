// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webassets

import (
	"io"
	"io/fs"
	"net/http"
	"os"
)

// ServeFile writes the specified file from the supplied fs.FS returning
// to the supplied writer, returning an appropriate http status code.
func ServeFile(wr io.Writer, fsys fs.FS, name string) (int, error) {
	f, err := fsys.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	if _, err := io.Copy(wr, f); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
