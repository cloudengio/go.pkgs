// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// PrefixReader is an io.ReadWriteCloser that prepends prefix to the data read
// from the underlying reader. It prepends the prefix to the beginning of the
// stream and after every separator character (defaults to newline). It is useful for prepending data
// to the output of an exec.Cmd without modifying the command itself when working
// with multiple outstanding commands.
type PrefixReader struct {
	prefix      []byte
	separator   rune
	r           *io.PipeReader
	w           *io.PipeWriter
	atLineStart bool
}

func NewPrefixReader(prefix []byte, separator rune) io.ReadWriteCloser {
	r, w := io.Pipe()
	return &PrefixReader{
		prefix:      prefix,
		separator:   separator,
		r:           r,
		w:           w,
		atLineStart: len(prefix) > 0,
	}
}

func (pr *PrefixReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	return pr.r.Read(p)
}

func (pr *PrefixReader) Write(p []byte) (n int, err error) {
	if len(pr.prefix) == 0 {
		return pr.w.Write(p)
	}

	originalLen := len(p)
	for len(p) > 0 {
		if pr.atLineStart {
			if _, err := pr.w.Write(pr.prefix); err != nil {
				return originalLen - len(p), err
			}
			pr.atLineStart = false
		}

		idx := bytes.IndexRune(p, pr.separator)
		if idx == -1 {
			// No separator, write the rest of the buffer.
			if _, err := pr.w.Write(p); err != nil {
				return originalLen - len(p), err
			}
			break
		}

		// Write up to and including the separator.
		nextIdx := idx + utf8.RuneLen(pr.separator)
		if _, err := pr.w.Write(p[:nextIdx]); err != nil {
			return originalLen - len(p), err
		}
		pr.atLineStart = true
		p = p[nextIdx:]
	}

	return originalLen, nil
}

func (pr *PrefixReader) Close() error {
	if err := pr.r.Close(); err != nil {
		return fmt.Errorf("failed to close read pipe: %w", err)
	}
	if err := pr.w.Close(); err != nil {
		return fmt.Errorf("failed to close write pipe: %w", err)
	}
	return nil
}
