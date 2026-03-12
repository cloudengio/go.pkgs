// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile will copy a local file with the option to overwrite an existing file
// and to set the permissions on the new file. It uses chmod to explicitly
// set permissions. It is not suitable for very large fles.
func CopyFile(from, to string, perms os.FileMode, overwrite bool) (returnErr error) {
	info, err := os.Stat(to)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("destination is a directory: %v", to)
		}
		if !overwrite {
			return fmt.Errorf("will not overwrite existing file: %v", to)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	output, err := os.OpenFile(to, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Close(); err != nil {
			// Return the error from the Close if the copy succeeded.
			if returnErr == nil {
				returnErr = err
			}
		}
		if err := os.Chmod(to, perms); err != nil {
			if returnErr == nil {
				returnErr = err
			}
		}
	}()
	input, err := os.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()
	_, returnErr = io.Copy(output, input)
	return
}

// IsDir returns true iff path exists and is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CopyAll will create an exact copy, including permissions, of a local
// filesystem hierarchy. The arguments must both refer to directories.
// A trailing slash (/) for the fromDir copies the contents of fromDir rather
// than fromDir itself. Thus:
//
//	CopyAll("a/b", "c") is the same as CopyAll("a/b/", "c/b")
//	and both create an exact copy of the tree a/b rooted at c/b.
//
// If overwrite is set any existing files will be overwritten. Existing
// directories will always have their contents updated.
// It uses os.Root scoped APIs to prevent symlink TOCTOU traversal.
func CopyAll(fromDir, toDir string, overwrite bool) error {
	for _, path := range []string{fromDir, toDir} {
		if !IsDir(path) {
			return fmt.Errorf("%v: not a directory", path)
		}
	}
	contents := strings.HasSuffix(fromDir, "/")
	topdir := filepath.Base(fromDir)

	fromRoot, err := os.OpenRoot(filepath.Clean(fromDir))
	if err != nil {
		return err
	}
	defer fromRoot.Close()

	toRoot, err := os.OpenRoot(toDir)
	if err != nil {
		return err
	}
	defer toRoot.Close()

	dstPath := func(p string) string {
		if contents {
			return p
		}
		return filepath.Join(topdir, p)
	}

	return fs.WalkDir(fromRoot.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			if !contents {
				return mkdirFromEntry(toRoot, topdir, d)
			}
			return nil
		}
		return copyEntry(fromRoot, toRoot, path, dstPath(path), d, overwrite)
	})
}

func mkdirFromEntry(root *os.Root, name string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	if err := root.Mkdir(name, info.Mode().Perm()); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func copyEntry(fromRoot, toRoot *os.Root, src, dst string, d fs.DirEntry, overwrite bool) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	if d.IsDir() {
		if err := toRoot.Mkdir(dst, info.Mode().Perm()); err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	}
	return copyFileRooted(fromRoot, toRoot, src, dst, info.Mode().Perm(), overwrite)
}

// copyFileRooted copies a file using root-scoped APIs to prevent symlink
// TOCTOU traversal.
func copyFileRooted(fromRoot, toRoot *os.Root, from, to string, perms os.FileMode, overwrite bool) (returnErr error) {
	info, err := toRoot.Stat(to)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("destination is a directory: %v", to)
		}
		if !overwrite {
			return fmt.Errorf("will not overwrite existing file: %v", to)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	output, err := toRoot.OpenFile(to, os.O_CREATE|os.O_RDWR|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Chmod(perms); err != nil {
			if returnErr == nil {
				returnErr = err
			}
		}
		if err := output.Close(); err != nil {
			if returnErr == nil {
				returnErr = err
			}
		}
	}()

	input, err := fromRoot.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()
	_, returnErr = io.Copy(output, input)
	return
}

// ListRegular returns the lexicographically ordered regular files that lie
// beneath dir. It uses os.Root scoped APIs to prevent symlink TOCTOU
// traversal.
func ListRegular(dir string) ([]string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var files []string
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == "." {
			return nil
		}
		if d.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ListDir returns the lexicographically ordered directories that lie beneath
// dir. It uses os.Root scoped APIs to prevent symlink TOCTOU traversal.
func ListDir(dir string) ([]string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var dirs []string
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == "." {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})
	return dirs, err
}
