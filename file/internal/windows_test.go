// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInaccessible(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test-file.text")
	if err := os.WriteFile(filename, []byte("hello world\n"), 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(filename); err != nil {
		t.Fatal(err)
	}
	if err := MakeInaccessibleToOwner(filename); err != nil {
		t.Fatal(err)
	}
	_, err := os.ReadFile(filename)
	if err == nil || !strings.Contains(err.Error(), "Access is denied") {
		t.Errorf("missing or incorrect error: %v", err)
	}

	err = os.WriteFile(filename, []byte("hello world\n"), 0666)
	if err == nil || !strings.Contains(err.Error(), "Access is denied") {
		t.Errorf("missing or incorrect error: %v", err)
	}

	dirname := filepath.Join(tmpdir, "test-dir", "sub-dir")
	if err := os.MkdirAll(dirname, 0777); err != nil {
		t.Fatal(err)
	}
}
