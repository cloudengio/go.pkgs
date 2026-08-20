// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cloudeng.io/os/lockedfile"
)

// ErrAlreadyRunning is returned when another orchestrator already holds the run
// lock.
var ErrAlreadyRunning = errors.New("another instance of this process is already running")

// UserConfigDirPath appends path to the per-user config directory as returned
// by os.UserConfigDir.
func UserConfigDirPath(path string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, path), nil
}

// AcquireRunLock takes a non-blocking, exclusive file lock so that only one
// orchestrator runs at a time. The returned unlock function must be called when
// the run finishes; the lock is also released automatically when the process
// exits — including on crash or SIGKILL — so it never goes stale. If another
// instance already holds the lock, ErrAlreadyRunning is returned, annotated with
// the holder's PID and the lock path.
//
// TryOpenFile is used rather than Mutex.TryLock so that the PID can be recorded
// through the locked handle itself. The Mutex's additional in-process guard is
// not needed: only RunCommand.Run acquires this lock, once per process.
func AcquireRunLock(path string) (unlock func(), err error) {

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, ok, err := lockedfile.TryOpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("acquiring run lock %s: %w", path, err)
	}
	if !ok {
		return nil, fmt.Errorf("another process is already running: pid: %s, lock file: %s: %w", lockHolder(path), path, ErrAlreadyRunning)
	}
	// Record our PID for diagnostics (best effort); truncate first in case a
	// previous holder wrote a longer value.
	_ = f.Truncate(0)
	_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	return func() { _ = f.Close() }, nil
}

// lockHolder returns a " (pid N)" suffix if the lock file records a PID.
func lockHolder(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	pid := strings.TrimSpace(string(data))
	if pid == "" {
		return ""
	}
	return " (pid " + pid + ")"
}
