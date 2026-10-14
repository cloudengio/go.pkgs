// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"os"
	"sync"
)

var chdirMu sync.Mutex

// ChdirLocked wraps os.Chdir with a global lock to ensure that changes to the
// current working directory are thread-safe since a process has only one
// working directory at a time. It returns a function that, when called,
// restores the original working directory and releases the lock, and returns
// the error, if any, from changing back to it; the lock is released whether or
// not that fails. The function may be called more than once, but only the first
// call has any effect, and returns that error: a later call returns nil and does
// not release a lock that another caller has since acquired.
func ChdirLocked(dir string) (func() error, error) {
	chdirMu.Lock()
	originalDir, err := os.Getwd()
	if err != nil {
		chdirMu.Unlock()
		return nil, err
	}
	if err := os.Chdir(dir); err != nil {
		chdirMu.Unlock()
		return nil, err
	}
	var once sync.Once
	return func() error {
		var err error
		once.Do(func() {
			defer chdirMu.Unlock()
			err = os.Chdir(originalDir)
		})
		return err
	}, nil
}
