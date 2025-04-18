// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lockedfile creates and manipulates files whose contents should only
// change atomically.
// This package is forked from the go compiler's internal packages.
package lockedfile

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"cloudeng.io/errors"
)

// A File is a locked *os.File.
//
// Closing the file releases the lock.
//
// If the program exits while a file is locked, the operating system releases
// the lock but may not do so promptly: callers must ensure that all locked
// files are closed before exiting.
type File struct {
	osFile
	closed bool
}

// osFile embeds a *os.File while keeping the pointer itself unexported.
// (When we close a File, it must be the same file descriptor that we opened!)
type osFile struct {
	*os.File
}

// OpenFile is like os.OpenFile, but returns a locked file.
// If flag includes os.O_WRONLY or os.O_RDWR, the file is write-locked;
// otherwise, it is read-locked.
func OpenFile(name string, flag int, perm os.FileMode) (*File, error) {
	var (
		f   = new(File)
		err error
	)
	f.File, err = openFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	// Although the operating system will drop locks for open files when the go
	// command exits, we want to hold locks for as little time as possible, and we
	// especially don't want to leave a file locked after we're done with it. Our
	// Close method is what releases the locks, so use a finalizer to report
	// missing Close calls on a best-effort basis.
	runtime.SetFinalizer(f, func(f *File) {
		panic(fmt.Sprintf("lockedfile.File %s became unreachable without a call to Close", f.Name()))
	})

	return f, nil
}

// Open is like os.Open, but returns a read-locked file.
func Open(name string) (*File, error) {
	return OpenFile(name, os.O_RDONLY, 0)
}

// Create is like os.Create, but returns a write-locked file.
func Create(name string) (*File, error) {
	return OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Edit creates the named file with mode 0666 (before umask),
// but does not truncate existing contents.
//
// If Edit succeeds, methods on the returned File can be used for I/O.
// The associated file descriptor has mode O_RDWR and the file is write-locked.
func Edit(name string) (*File, error) {
	return OpenFile(name, os.O_RDWR|os.O_CREATE, 0666)
}

// Close unlocks and closes the underlying file.
//
// Close may be called multiple times; all calls after the first will return a
// non-nil error.
func (f *File) Close() error {
	if f.closed {
		return &os.PathError{
			Op:   "close",
			Path: f.Name(),
			Err:  os.ErrClosed,
		}
	}
	f.closed = true

	err := closeFile(f.File)
	runtime.SetFinalizer(f, nil)
	return err
}

// Read opens the named file with a read-lock and returns its contents.
func Read(name string) ([]byte, error) {
	f, err := Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// Write opens the named file (creating it with the given permissions if needed),
// then write-locks it and overwrites it with the given content.
func Write(name string, content io.Reader, perm os.FileMode) (err error) {
	f, err := OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, content)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Transform invokes t with the result of reading the named file, with its lock
// still held.
//
// If t returns a nil error, Transform then writes the returned contents back to
// the file, making a best effort to preserve existing contents on error.
//
// t must not modify the slice passed to it.
func Transform(name string, t func([]byte) ([]byte, error)) (err error) {
	f, err := Edit(name)
	if err != nil {
		return err
	}
	defer f.Close()

	old, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	newData, err := t(old)
	if err != nil {
		return err
	}

	if len(newData) > len(old) {
		// The overall file size is increasing, so write the tail first: if we're
		// about to run out of space on the disk, we would rather detect that
		// failure before we have overwritten the original contents.
		if _, err := f.WriteAt(newData[len(old):], int64(len(old))); err != nil {
			// Make a best effort to remove the incomplete tail.
			if nerr := f.Truncate(int64(len(old))); nerr != nil {
				return errors.NewM(err, nerr)
			}
			return err
		}
	}

	// We're about to overwrite the old contents. In case of failure, make a best
	// effort to roll back before we close the file.
	defer func() {
		if err != nil {
			if _, err := f.WriteAt(old, 0); err == nil {
				_ = f.Truncate(int64(len(old)))
			}
		}
	}()

	if len(newData) >= len(old) {
		if _, err := f.WriteAt(newData[:len(old)], 0); err != nil {
			return err
		}
	} else {
		if _, err := f.WriteAt(newData, 0); err != nil {
			return err
		}
		// The overall file size is decreasing, so shrink the file to its final size
		// after writing. We do this after writing (instead of before) so that if
		// the write fails, enough filesystem space will likely still be reserved
		// to contain the previous contents.
		if err := f.Truncate(int64(len(newData))); err != nil {
			return err
		}
	}

	return nil
}
