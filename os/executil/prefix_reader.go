// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"fmt"
	"io"
)

// PrefixReader is an io.ReadWriteCloser that prepends prefix to the data read
// from the underlying reader. It is useful for prepending data to the
// output of an exec.Cmd without modifying the command itself when working
// with multiple outstanding commands.
type PrefixReader struct {
	prefix []byte
	r      *io.PipeReader
	w      *io.PipeWriter
}

func NewPrefixReader(prefix []byte) io.ReadWriteCloser {
	r, w := io.Pipe()
	return &PrefixReader{prefix: prefix, r: r, w: w}
}

func (r *PrefixReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	pn := copy(p, r.prefix)
	p = p[pn:]
	if pn < len(r.prefix) {
		return pn, nil
	}
	n, err = r.r.Read(p)
	return pn + n, err
}

func (r *PrefixReader) Write(p []byte) (n int, err error) {
	return r.w.Write(p)
}

func (r *PrefixReader) Close() error {
	if err := r.r.Close(); err != nil {
		return fmt.Errorf("failed to close read pipe: %w", err)
	}
	if err := r.w.Close(); err != nil {
		return fmt.Errorf("failed to close write pipe: %w", err)
	}
	return nil
}
