// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/errors"
	"cloudeng.io/windows/win32testutil"
)

func CreateTestTree() string {
	tmpDir, err := os.MkdirTemp("", "filewalk")
	if err != nil {
		fmt.Printf("failed to create testdir: %v", err)
		os.RemoveAll(tmpDir)
		os.Exit(0)
	}
	if err := createTestDir(tmpDir); err != nil {
		fmt.Printf("failed to create testdir: %v", err)
		os.RemoveAll(tmpDir)
		os.Exit(0)
	}
	return tmpDir
}

func createTestDir(tmpDir string) error {
	j := filepath.Join
	errs := errors.M{}
	dirs := []string{
		j("a0"),
		j("a0", "a0.0"),
		j("a0", "a0.1"),
		j("b0", "b0.0"),
		j("b0", "b0.1", "b1.0"),
	}
	for _, dir := range append([]string{""}, dirs...) {
		err := os.MkdirAll(j(tmpDir, dir), 0777)
		errs.Append(err)
		for _, file := range []string{"f0", "f1", "f2"} {
			err = os.WriteFile(j(tmpDir, dir, file), []byte{'1', '2', '3'}, 0666) // #nosec G306
			errs.Append(err)
		}
	}
	err := os.Mkdir(j(tmpDir, "inaccessible-dir"), 0000)
	errs.Append(err)
	err = win32testutil.MakeInaccessibleToOwner(j(tmpDir, "inaccessible-dir"))
	errs.Append(err)
	err = os.Symlink(j("a0", "f0"), j(tmpDir, "lf0"))
	errs.Append(err)
	err = os.Symlink(j("a0"), j(tmpDir, "la0"))
	errs.Append(err)
	err = os.Symlink("nowhere", j(tmpDir, "la1"))
	errs.Append(err)
	err = os.WriteFile(j(tmpDir, "a0", "inaccessible-file"), []byte{'1', '2', '3'}, 0000)
	errs.Append(err)
	err = win32testutil.MakeInaccessibleToOwner(j(tmpDir, "a0", "inaccessible-file")) // windows.
	errs.Append(err)
	return errs.Err()
}
