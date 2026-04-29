// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bytes"
	"testing"

	"cloudeng.io/os/executil"
)

func TestLabelingWriter_EmptyWriteBuffer(t *testing.T) {
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, []byte(">> "), '\n')

	n, err := w.Write([]byte{})
	if n != 0 || err != nil {
		t.Errorf("Write([]byte{}): got (%d, %v), want (0, nil)", n, err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestLabelingWriter_EmptyPrefix(t *testing.T) {
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, nil, '\n')

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 5 {
		t.Errorf("Write returned %d bytes, want 5", n)
	}
	if got := buf.String(); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestLabelingWriter_NewlineInsertion(t *testing.T) {
	tag := ">> "
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, []byte(tag), '\n')

	// Write pieces of string
	_, _ = w.Write([]byte("line 1\n"))
	_, _ = w.Write([]byte("line "))
	_, _ = w.Write([]byte("2\nlin"))
	_, _ = w.Write([]byte("e 3"))
	_, _ = w.Write([]byte("\n"))

	got := buf.String()
	want := ">> line 1\n>> line 2\n>> line 3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLabelingWriter_NoTrailingNewline(t *testing.T) {
	tag := ">> "
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, []byte(tag), '\n')

	_, _ = w.Write([]byte("line 1\nline 2"))

	got := buf.String()
	want := ">> line 1\n>> line 2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLabelingWriter_TabSeparator(t *testing.T) {
	tag := "-> "
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, []byte(tag), '\t')

	_, _ = w.Write([]byte("col A\tcol "))
	_, _ = w.Write([]byte("B\tco"))
	_, _ = w.Write([]byte("l C"))

	got := buf.String()
	want := "-> col A\t-> col B\t-> col C"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLabelingWriter_WriteCounts(t *testing.T) {
	tag := ">> "
	var buf bytes.Buffer
	w := executil.NewLabelingWriter(&buf, []byte(tag), '\n')

	writes := []string{
		"line 1\n",
		"line 2",
		"\nline 3\n",
		"line 4",
	}

	for _, str := range writes {
		n, err := w.Write([]byte(str))
		if err != nil {
			t.Fatalf("Write(%q) failed: %v", str, err)
		}
		if n != len(str) {
			t.Errorf("Write(%q) returned %d bytes, want %d", str, n, len(str))
		}
	}

	got := buf.String()
	want := ">> line 1\n>> line 2\n>> line 3\n>> line 4"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
