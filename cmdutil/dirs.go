package cmdutil

import (
	"fmt"
	"io"
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
	} else {
		if !os.IsNotExist(err) {
			return err
		}
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
//   CopyAll("a/b", "c") is the same as CopyAll("a/b/", "c/b")
//   and both create an exact copy of the tree a/b rooted at c/b.
// If overwrite is set any existing files will be overwritten. Existing
// directories will always have their contents updated.
// It is not intended for use with very large directory trees since it uses
// filepath.Walk.
func CopyAll(fromDir, toDir string, ovewrite bool) error {
	for _, path := range []string{fromDir, toDir} {
		if !IsDir(path) {
			return fmt.Errorf("%v: not a directory", path)
		}
	}
	contents := strings.HasSuffix(fromDir, "/")
	topdir := filepath.Base(fromDir)
	toPath := func(p string) string {
		if contents {
			return filepath.Join(toDir, strings.TrimPrefix(p, topdir))
		}
		return filepath.Join(toDir, p)
	}
	return filepath.Walk(fromDir, func(path string, info os.FileInfo, err error) error {
		if contents && (fromDir == path) {
			return nil
		}
		if err != nil {
			return err
		}
		dst := toPath(path)
		if info.IsDir() {
			if err := os.Mkdir(dst, info.Mode().Perm()); err != nil && !os.IsExist(err) {
				return err
			}
			return nil
		}
		return CopyFile(
			path,
			dst,
			info.Mode().Perm(),
			ovewrite,
		)
	})
}
