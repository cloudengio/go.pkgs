// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllocated(t *testing.T) {
	td := t.TempDir()
	filename := filepath.Join(td, "testfile")
	size := int64(1024*1024*1) + 33 // 1 MB + 33 bytes
	buf := make([]byte, size)
	if err := os.WriteFile(filename, buf, 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", filename, err)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open file %s: %v", filename, err)
	}
	defer f.Close()
	ok, err := allocated(f, size)
	if err != nil {
		t.Fatalf("failed to check allocation for file %s: %v", filename, err)
	}
	if !ok {
		t.Errorf("file %s was not allocated with size %d", filename, size)
	}

}
