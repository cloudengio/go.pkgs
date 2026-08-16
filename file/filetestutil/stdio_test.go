// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"cloudeng.io/file/filetestutil"
)

func TestCaptureStdout(t *testing.T) {
	out, err := filetestutil.CaptureStdout(func() error {
		fmt.Print("hello stdout")
		return nil
	})
	if err != nil {
		t.Fatalf("CaptureStdout: %v", err)
	}
	if string(out) != "hello stdout" {
		t.Errorf("got %q, want %q", string(out), "hello stdout")
	}

	testErr := errors.New("test error")
	out2, err2 := filetestutil.CaptureStdout(func() error {
		fmt.Print("with error")
		return testErr
	})
	if !errors.Is(err2, testErr) {
		t.Errorf("got %v, want %v", err2, testErr)
	}
	if string(out2) != "with error" {
		t.Errorf("got %q, want %q", string(out2), "with error")
	}
}

func TestCaptureStderr(t *testing.T) {
	out, err := filetestutil.CaptureStderr(func() error {
		fmt.Fprint(os.Stderr, "hello stderr")
		return nil
	})
	if err != nil {
		t.Fatalf("CaptureStderr: %v", err)
	}
	if string(out) != "hello stderr" {
		t.Errorf("got %q, want %q", string(out), "hello stderr")
	}
}

func TestFeedStdin(t *testing.T) {
	var readData string
	err := filetestutil.FeedStdin("hello stdin", func() error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		readData = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("FeedStdin: %v", err)
	}
	if readData != "hello stdin" {
		t.Errorf("got %q, want %q", readData, "hello stdin")
	}
}

func TestCaptureStdoutPanic(t *testing.T) {
	origStdout := os.Stdout
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if os.Stdout != origStdout {
			t.Errorf("os.Stdout was not restored: got %v, want %v", os.Stdout, origStdout)
		}
	}()

	_, _ = filetestutil.CaptureStdout(func() error {
		fmt.Print("something before panic")
		panic("panic in stdout capture")
	})
}

func TestCaptureStderrPanic(t *testing.T) {
	origStderr := os.Stderr
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if os.Stderr != origStderr {
			t.Errorf("os.Stderr was not restored: got %v, want %v", os.Stderr, origStderr)
		}
	}()

	_, _ = filetestutil.CaptureStderr(func() error {
		fmt.Fprint(os.Stderr, "something before panic")
		panic("panic in stderr capture")
	})
}

func TestFeedStdinPanic(t *testing.T) {
	origStdin := os.Stdin
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if os.Stdin != origStdin {
			t.Errorf("os.Stdin was not restored: got %v, want %v", os.Stdin, origStdin)
		}
	}()

	_ = filetestutil.FeedStdin("hello", func() error {
		panic("panic in stdin feed")
	})
}
