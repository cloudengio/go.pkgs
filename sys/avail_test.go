// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package sys_test

import (
	"os"
	"testing"

	"cloudeng.io/sys"
)

func TestStatfs(t *testing.T) {
	avail, err := sys.AvailableBytes(os.TempDir())
	if err != nil {
		t.Fatalf("failed to get available bytes: %v", err)
	}
	if avail <= 0 {
		t.Fatalf("expected positive available bytes, got %d", avail)
	}
	t.Logf("Available bytes in %s: %d", os.TempDir(), avail)
}
