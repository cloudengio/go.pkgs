// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bytes"
	"testing"

	"cloudeng.io/os/executil"
)

func TestTailWriter(t *testing.T) {
	tests := []struct {
		desc  string
		size  int
		write []string
		want  string
	}{
		{
			desc:  "write less than capacity",
			size:  10,
			write: []string{"hello"},
			want:  "hello",
		},
		{
			desc:  "write exactly capacity",
			size:  5,
			write: []string{"hello"},
			want:  "hello",
		},
		{
			desc:  "write slightly more than capacity",
			size:  5,
			write: []string{"hello", "!"},
			want:  "ello!",
		},
		{
			desc:  "write far more than capacity",
			size:  5,
			write: []string{"abcdefghijklmnopqrstuvwxyz"},
			want:  "vwxyz",
		},
		{
			desc:  "multiple writes",
			size:  10,
			write: []string{"hello, ", "world!"}, // 13 chars total
			want:  "lo, world!",                  // 10 chars
		},
		{
			desc:  "zero size",
			size:  0,
			write: []string{"hello"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			w := executil.NewTailWriter(tt.size)
			for _, s := range tt.write {
				n, err := w.Write([]byte(s))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if n != len(s) {
					t.Fatalf("wrote %d bytes, expected %d", n, len(s))
				}
			}

			if tt.size == 0 {
				if len(w.Bytes()) != 0 {
					t.Errorf("got %q, want \"\"", w.Bytes())
				}
				return
			}

			if string(w.Bytes()) != tt.want {
				t.Errorf("got %q, want %q", w.Bytes(), tt.want)
			}
		})
	}
}

func TestTailWriterConcurrent(t *testing.T) {
	w := executil.NewTailWriter(100)
	done := make(chan bool)
	go func() {
		for i := 0; i < 1000; i++ {
			_, _ = w.Write([]byte("a"))
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			_, _ = w.Write([]byte("b"))
		}
		done <- true
	}()
	<-done
	<-done
	got := w.Bytes()
	if len(got) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(got))
	}
	if !bytes.Contains(got, []byte("a")) && !bytes.Contains(got, []byte("b")) {
		t.Errorf("expected a mix of a and b")
	}
}
