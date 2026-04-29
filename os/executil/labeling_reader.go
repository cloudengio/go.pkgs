// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"bytes"
	"io"
	"sync"
	"unicode/utf8"
)

// LabelingWriter is an io.Writer that prepends prefix to the data written
// from the underlying writer. It prepends the prefix to the beginning of the
// stream and after every separator character. For example, it can be used
// to insert labels in the output of an exec.Cmd without modifying the command
// itself when working with multiple outstanding commands.
type LabelingWriter struct {
	mu          sync.Mutex
	prefix      []byte
	separator   rune
	w           io.Writer
	atLineStart bool
}

func NewLabelingWriter(w io.Writer, prefix []byte, separator rune) io.Writer {
	return &LabelingWriter{
		prefix:      prefix,
		separator:   separator,
		w:           w,
		atLineStart: len(prefix) > 0,
	}
}

// Write implements io.Writer. It writes the data to the underlying writer,
// inserting the prefix at the beginning of the stream and after every separator
// character. It returns the number of bytes from p that were written
// rather than the total number of bytes including the label.
func (pr *LabelingWriter) Write(p []byte) (n int, err error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

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
			if n, err := pr.w.Write(p); err != nil {
				return originalLen - len(p) + n, err
			}
			break
		}

		// Write up to and including the separator.
		nextIdx := idx + utf8.RuneLen(pr.separator)
		if n, err := pr.w.Write(p[:nextIdx]); err != nil {
			return originalLen - len(p) + n, err
		}
		pr.atLineStart = true
		p = p[nextIdx:]
	}

	return originalLen, nil
}
