// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"cloudeng.io/os/executil"
)

func asyncWrite(rw io.ReadWriteCloser, p []byte) <-chan error {
	ch := make(chan error, 1)
	go func() {
		var err error
		if len(p) > 0 {
			_, err = rw.Write(p)
		}
		ch <- err
	}()
	return ch
}

func TestPrefixReader_EmptyReadBuffer(t *testing.T) {
	pr := executil.NewPrefixReader([]byte(">> "), '\n')
	defer pr.Close()

	n, err := pr.Read([]byte{})
	if n != 0 || err != nil {
		t.Errorf("Read([]byte{}): got (%d, %v), want (0, nil)", n, err)
	}
}

func TestPrefixReader_EmptyTag(t *testing.T) {
	pr := executil.NewPrefixReader(nil, '\n')
	defer pr.Close()
	werr := asyncWrite(pr, []byte("hello"))

	buf := make([]byte, 20)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got, want := string(buf[:n]), "hello"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := <-werr; err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestPrefixReader_NewlineInsertion(t *testing.T) {
	tag := ">> "
	pr := executil.NewPrefixReader([]byte(tag), '\n')
	defer pr.Close()

	reads := make(chan string)
	go func() {
		var buf bytes.Buffer
		// reads everything until EOF (pipe closed)
		_, err := io.Copy(&buf, pr)
		if err != nil && !strings.Contains(err.Error(), "read/write on closed pipe") {
			t.Errorf("Copy error: %v", err)
		}
		reads <- buf.String()
	}()

	// Write pieces of string
	pr.Write([]byte("line 1\n"))
	pr.Write([]byte("line "))
	pr.Write([]byte("2\nlin"))
	pr.Write([]byte("e 3"))
	pr.Write([]byte("\n"))
	pr.Close()

	got := <-reads
	want := ">> line 1\n>> line 2\n>> line 3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixReader_NoTrailingNewline(t *testing.T) {
	tag := ">> "
	pr := executil.NewPrefixReader([]byte(tag), '\n')
	defer pr.Close()

	reads := make(chan string)
	go func() {
		out, err := io.ReadAll(pr)
		if err != nil && !strings.Contains(err.Error(), "read/write on closed pipe") {
			t.Errorf("ReadAll error: %v", err)
		}
		reads <- string(out)
	}()

	pr.Write([]byte("line 1\nline 2"))
	pr.Close()

	got := <-reads
	want := ">> line 1\n>> line 2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixReader_TabSeparator(t *testing.T) {
	tag := "-> "
	pr := executil.NewPrefixReader([]byte(tag), '\t')
	defer pr.Close()

	reads := make(chan string)
	go func() {
		out, err := io.ReadAll(pr)
		if err != nil && !strings.Contains(err.Error(), "read/write on closed pipe") {
			t.Errorf("ReadAll error: %v", err)
		}
		reads <- string(out)
	}()

	pr.Write([]byte("col A\tcol "))
	pr.Write([]byte("B\tco"))
	pr.Write([]byte("l C"))
	pr.Close()

	got := <-reads
	want := "-> col A\t-> col B\t-> col C"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
