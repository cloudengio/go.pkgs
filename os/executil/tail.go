// Copyright 2026    cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"slices"
	"sync"
)

// TailWriter is an io.Writer that keeps only the last N bytes
// written to it. It is useful for capturing the output of a
// command while limiting memory usage.
type TailWriter struct {
	mu   sync.Mutex
	buf  []byte
	size int
	pos  int // current write position
	full bool
}

// NewTailWriter creates a new TailWriter that keeps the last n bytes.
func NewTailWriter(n int) *TailWriter {
	return &TailWriter{
		buf:  make([]byte, n),
		size: n,
	}
}

// Write implements the io.Writer interface.
func (w *TailWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n = len(p)
	if w.size == 0 {
		return n, nil
	}
	if len(p) > w.size {
		p = p[len(p)-w.size:]
	}
	for len(p) > 0 {
		chunk := copy(w.buf[w.pos:], p)
		w.pos += chunk
		if w.pos == w.size {
			w.pos = 0
			w.full = true
		}
		p = p[chunk:]
	}
	return n, nil
}

// Bytes returns the contents of the TailWriter.
func (w *TailWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.full {
		result := make([]byte, w.pos)
		copy(result, w.buf[:w.pos])
		return slices.Clone(result)
	}
	// Return rotated buffer
	if w.size == 0 {
		return nil
	}
	result := make([]byte, w.size)
	copy(result, w.buf[w.pos:])
	copy(result[w.size-w.pos:], w.buf[:w.pos])
	return slices.Clone(result)
}
