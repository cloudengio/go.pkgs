// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package os

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// SignalAndWait provides a convenience function to signal a process
// to terminate by sending it one or more signals and waiting for
// it to terminate but with a timeout on calling Wait and
// on waiting for the process to stop after each signal.
// The perSignalOrWait duration is used as the timeout for both
// calling Wait and for waiting for the process to stop after
// each signal, hence the total time spent waiting may be
// up to len(sigs)+1 times perSignalOrWait.
// If the process stops after any signal, SignalAndWait returns
// immediately.
func SignalAndWait(ctx context.Context, perSignalOrWait time.Duration, cmd *exec.Cmd, sigs ...os.Signal) error {
	pid := cmd.Process.Pid

	doneCh := make(chan struct{})
	go func() {
		cmd.Wait()
		close(doneCh)
	}()
	wait := true

	for _, sig := range sigs {
		if err := cmd.Process.Signal(sig); err != nil {
			return err
		}
		if wait {
			ctx, cancel := context.WithTimeout(ctx, perSignalOrWait)
			select {
			case <-doneCh:
			case <-ctx.Done():
			}
			cancel()
			wait = false
		}
		if err := WaitForStopped(ctx, pid, perSignalOrWait); err == nil {
			return nil
		}
	}
	return WaitForStopped(ctx, pid, perSignalOrWait)
}

// IsStopped returns true if the process with the specified pid has
// stopped or does not exist. Wait must have been called on the
// process otherwise this function will return true on some systems
// since the process may still exist as a defunct process.
func IsStopped(pid int) bool {
	return isStopped(pid)
}

// WaitForStopped waits for the process with the specified pid to stop
// within the specified duration. It assumes that Wait has been called.
func WaitForStopped(ctx context.Context, pid int, waitFor time.Duration) error {
	if isStopped(pid) {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, waitFor)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			if isStopped(pid) {
				return nil
			}
		}
	}
}
