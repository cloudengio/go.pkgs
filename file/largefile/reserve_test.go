// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"os"
	"path/filepath"
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
}
