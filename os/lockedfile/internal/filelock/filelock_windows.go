// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package filelock

import (
	"os"
	"syscall"
)

type lockType uint32

const (
	readLock  lockType = 0
	writeLock lockType = LOCKFILE_EXCLUSIVE_LOCK
)

const (
	reserved = 0
	allBytes = ^uint32(0)
)

// When LockFileEx is called with LOCKFILE_FAIL_IMMEDIATELY and the region is
// already locked, it fails immediately with ERROR_LOCK_VIOLATION, or with
// ERROR_SHARING_VIOLATION if another process holds it with an incompatible
// sharing mode.
const (
	ERROR_SHARING_VIOLATION syscall.Errno = 32
	ERROR_LOCK_VIOLATION    syscall.Errno = 33
)

func tryLock(f File, lt lockType) (bool, error) {
	ol := new(syscall.Overlapped)
	err := LockFileEx(syscall.Handle(f.Fd()), uint32(lt)|LOCKFILE_FAIL_IMMEDIATELY, reserved, allBytes, allBytes, ol)
	if err == nil {
		return true, nil
	}
	if err == ERROR_LOCK_VIOLATION || err == ERROR_SHARING_VIOLATION {
		return false, nil
	}
	return false, &os.PathError{
		Op:   "Try" + lt.String(),
		Path: f.Name(),
		Err:  err,
	}
}

func lock(f File, lt lockType) error {
	// Per https://golang.org/issue/19098, “Programs currently expect the Fd
	// method to return a handle that uses ordinary synchronous I/O.”
	// However, LockFileEx still requires an OVERLAPPED structure,
	// which contains the file offset of the beginning of the lock range.
	// We want to lock the entire file, so we leave the offset as zero.
	ol := new(syscall.Overlapped)

	err := LockFileEx(syscall.Handle(f.Fd()), uint32(lt), reserved, allBytes, allBytes, ol)
	if err != nil {
		return &os.PathError{
			Op:   lt.String(),
			Path: f.Name(),
			Err:  err,
		}
	}
	return nil
}

func unlock(f File) error {
	ol := new(syscall.Overlapped)
	err := UnlockFileEx(syscall.Handle(f.Fd()), reserved, allBytes, allBytes, ol)
	if err != nil {
		return &os.PathError{
			Op:   "Unlock",
			Path: f.Name(),
			Err:  err,
		}
	}
	return nil
}

func isNotSupported(err error) bool {
	switch err {
	case ERROR_NOT_SUPPORTED, ERROR_CALL_NOT_IMPLEMENTED, ErrNotSupported:
		return true
	default:
		return false
	}
}
