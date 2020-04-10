package cmdutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile will copy a single, local filesystem, file with the option to
// overwrite an existing file and to set the permissions on the new file.
func CopyFile(from, to string, perms os.FileMode, overwrite bool) (returnErr error) {
	info, err := os.Stat(to)
	if exists := err == nil; exists {
		if !overwrite {
			return fmt.Errorf("will not overwrite existing file: %v", to)
		}
		if info.IsDir() {
			return fmt.Errorf("destination is a directory: %v", to)
		}
	}
	if err != nil {
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
			// return the error from the Close if the copy succeeded.
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

// CopyAll will create an exact copy, including permissions, of all of the
// directories and files under fromDir in toDir. If overwrite is set any existing
// files will be overwritten.
func CopyAll(fromDir, toDir string, ovewrite bool) error {
	for _, path := range []string{fromDir, toDir} {
		if !IsDir(path) {
			return fmt.Errorf("%v: not a directory", path)
		}
	}
	return filepath.Walk(fromDir, func(path string, info os.FileInfo, err error) error {
		if fromDir == path {
			return nil
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := os.Mkdir(filepath.Join(toDir, path), info.Mode().Perm()); err != nil && !os.IsExist(err) {
				return err
			}
			return nil
		}
		return CopyFile(
			filepath.Join(fromDir, path),
			filepath.Join(toDir, path),
			info.Mode().Perm(),
			ovewrite,
		)
	})
}
