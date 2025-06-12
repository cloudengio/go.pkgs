// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestReserveSpace(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()
	filename := filepath.Join(td, "testfile")
	os.Remove(filename) // Ensure the file does not exist before the test

	size := int64(1024*1024*10) + 33 // 10 MB
	size = 30
	if err := ReserveSpace(ctx, filename, size, 4096, 4); err != nil {
		t.Fatalf("%v: %v", filename, err)
	}
	t.Fail()

	fi, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to stat file %s: %v", filename, err)
	}
	s := fi.Sys().(*syscall.Stat_t)
	fmt.Printf("File %s: Size %d, Blocks %d, Blksize %d\n", filename, fi.Size(), s.Blocks, s.Blksize)
	ns := s.Blocks * int64(s.Blksize)
	fmt.Printf("%v -- %v (%v)\n", size, ns, ns/size)
}
