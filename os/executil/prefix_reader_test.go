// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"io"
	"testing"

	"cloudeng.io/os/executil"
)

// asyncWrite writes p to rw in a goroutine and returns a channel that receives
// the write error (or nil) when the write completes. Because PrefixReader uses
// an unbuffered pipe, Write blocks until the data is consumed by a Read.
func asyncWrite(rw io.ReadWriteCloser, p []byte) <-chan error {
	ch := make(chan error, 1)
	go func() {
		_, err := rw.Write(p)
		ch <- err
	}()
	return ch
}

func TestPrefixReader_EmptyReadBuffer(t *testing.T) {
	pr := executil.NewPrefixReader([]byte(">> "))
	defer pr.Close()

	n, err := pr.Read([]byte{})
	if n != 0 || err != nil {
		t.Errorf("Read([]byte{}): got (%d, %v), want (0, nil)", n, err)
	}
}

func TestPrefixReader_BasicRead(t *testing.T) {
	pr := executil.NewPrefixReader([]byte(">> "))
	defer pr.Close()

	werr := asyncWrite(pr, []byte("hello"))

	buf := make([]byte, 20)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got, want := string(buf[:n]), ">> hello"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := <-werr; err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestPrefixReader_EmptyTag(t *testing.T) {
	pr := executil.NewPrefixReader(nil)
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

// TestPrefixReader_BufferSmallerThanTag verifies that when the read buffer is
// too small to hold the tag, Read returns a partial tag without consuming any
// data from the pipe. The subsequent Read restarts with the full tag.
func TestPrefixReader_BufferSmallerThanTag(t *testing.T) {
	tag := ">> " // 3 bytes
	pr := executil.NewPrefixReader([]byte(tag))
	defer pr.Close()

	werr := asyncWrite(pr, []byte("hello"))

	// Buffer holds only 2 bytes — smaller than the 3-byte tag.
	// Read must return the partial tag without touching the pipe.
	buf := make([]byte, 2)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("first Read (small buffer): %v", err)
	}
	if got, want := string(buf[:n]), ">>"; got != want {
		t.Errorf("partial tag: got %q, want %q", got, want)
	}

	// The write goroutine is still blocked: the pipe data was not consumed.
	// A subsequent Read with a large-enough buffer gets the full tag + data.
	buf2 := make([]byte, 20)
	n2, err := pr.Read(buf2)
	if err != nil {
		t.Fatalf("second Read (large buffer): %v", err)
	}
	if got, want := string(buf2[:n2]), ">> hello"; got != want {
		t.Errorf("full read: got %q, want %q", got, want)
	}
	if err := <-werr; err != nil {
		t.Errorf("Write: %v", err)
	}
}

// TestPrefixReader_BufferExactlyTagSize verifies that when the buffer holds
// exactly the tag, Read fills it with the tag and returns without consuming
// pipe data (because the remaining slice has zero length).
func TestPrefixReader_BufferExactlyTagSize(t *testing.T) {
	tag := ">> " // 3 bytes
	pr := executil.NewPrefixReader([]byte(tag))
	defer pr.Close()

	werr := asyncWrite(pr, []byte("hello"))

	buf := make([]byte, len(tag))
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read (tag-sized buffer): %v", err)
	}
	if got := string(buf[:n]); got != tag {
		t.Errorf("got %q, want %q", got, tag)
	}

	// Write goroutine is still blocked; consume its data now.
	buf2 := make([]byte, 20)
	n2, err := pr.Read(buf2)
	if err != nil {
		t.Fatalf("second Read: %v", err)
	}
	if got, want := string(buf2[:n2]), ">> hello"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := <-werr; err != nil {
		t.Errorf("Write: %v", err)
	}
}

// TestPrefixReader_TagPrependedOnEachRead verifies that every call to Read
// prepends the tag, regardless of how many reads have occurred.
func TestPrefixReader_TagPrependedOnEachRead(t *testing.T) {
	tag := "[TAG] "
	pr := executil.NewPrefixReader([]byte(tag))
	defer pr.Close()

	writes := []string{"first line", "second line"}
	// Writes are sequential because each Write blocks until its data is read.
	go func() {
		for _, s := range writes {
			pr.Write([]byte(s)) //nolint:errcheck
		}
	}()

	buf := make([]byte, 64)
	for i, want := range []string{
		tag + "first line",
		tag + "second line",
	} {
		n, err := pr.Read(buf)
		if err != nil {
			t.Fatalf("Read[%d]: %v", i, err)
		}
		if got := string(buf[:n]); got != want {
			t.Errorf("Read[%d]: got %q, want %q", i, got, want)
		}
	}
}
