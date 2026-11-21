// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !unix

package pssh

import (
	"io"
	"os"
)

// pumpStdin copies f to w until f is at EOF. It cannot be interrupted on this
// platform and so a read of f may remain outstanding once stop is closed.
func pumpStdin(f *os.File, w io.WriteCloser, _ <-chan struct{}) {
	_, _ = io.Copy(w, f)
	_ = w.Close()
}

// watchResize is not supported on this platform.
func watchResize(*os.File, <-chan struct{}) <-chan windowSize {
	return nil
}
