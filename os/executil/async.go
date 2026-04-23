// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"os/exec"
	"sync"
)

// AsyncWait simplifies implementing asynchronous waiting for an exec.Cmd.
type AsyncWait struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	err    error
	done   bool
	doneCh chan struct{}
}

// NewAsyncWait creates a new AsyncWait for the given exec.Cmd.
// It immediately starts a goroutine to wait for the cmd to complete.
func NewAsyncWait(cmd *exec.Cmd) *AsyncWait {
	aw := &AsyncWait{
		cmd:    cmd,
		doneCh: make(chan struct{}),
	}
	aw.runWait()
	return aw
}

func (aw *AsyncWait) runWait() {
	go func() {
		err := aw.cmd.Wait()
		aw.mu.Lock()
		aw.err = err
		aw.done = true
		close(aw.doneCh)
		aw.mu.Unlock()
	}()
}

// WaitDone reports whether the cmd has already completed.
// If so, it returns true and the error from cmd.Wait(), otherwise
// it returns false and nil.
func (aw *AsyncWait) WaitDone() (bool, error) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.done, aw.err
}

// Wait waits for the cmd to complete and returns the error from cmd.Wait().
// If the cmd has already completed,
// it returns immediately with the error from cmd.Wait().
func (aw *AsyncWait) Wait() error {
	<-aw.doneCh
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.err
}

// Cmd returns the exec.Cmd being waited on.
func (aw *AsyncWait) Cmd() *exec.Cmd {
	return aw.cmd
}
