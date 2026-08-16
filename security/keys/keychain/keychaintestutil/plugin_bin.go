// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"cloudeng.io/os/executil"
)

// BuildPluginBinary compiles the in-memory test plugin executable into a temporary directory
// and registers its cleanup with t.Cleanup. It returns the absolute path to the binary.
func BuildPluginBinary(t testing.TB) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller information")
	}
	srcDir := filepath.Join(filepath.Dir(currentFile), "cmd", "keychain-test-plugin")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "keychain-test-plugin")

	path, err := executil.GoBuild(context.Background(), binPath, srcDir)
	if err != nil {
		t.Fatalf("failed to build test plugin binary: %v", err)
	}
	return path
}
