// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package pssh

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// pollInterval is how often pumpStdin looks to see if it has been stopped.
const pollInterval = 100 // milliseconds

// pumpStdin copies f to w until f is at EOF or stop is closed. Unlike io.Copy
// it does not leave a read of f outstanding when stopped, which would consume
// the next input intended for something else.
func pumpStdin(f *os.File, w io.WriteCloser, stop <-chan struct{}) {
	fd := int(f.Fd()) //nolint:gosec // fds fit in an int.
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-stop:
			return
		default:
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}} //nolint:gosec
		n, err := unix.Poll(fds, pollInterval)
		if errors.Is(err, unix.EINTR) || (err == nil && n == 0) {
			continue
		}
		if err != nil {
			return
		}
		m, err := f.Read(buf)
		if m > 0 {
			if _, werr := w.Write(buf[:m]); werr != nil {
				return
			}
		}
		if err != nil {
			_ = w.Close()
			return
		}
	}
}

// watchResize returns the sizes of f as it changes, until stop is closed.
func watchResize(f *os.File, stop <-chan struct{}) <-chan windowSize {
	if f == nil {
		return nil
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)
	sizes := make(chan windowSize)
	go func() {
		defer signal.Stop(sigs)
		for {
			select {
			case <-stop:
				return
			case <-sigs:
			}
			w, h, err := term.GetSize(int(f.Fd())) //nolint:gosec
			if err != nil {
				continue
			}
			select {
			case sizes <- windowSize{w, h}:
			case <-stop:
				return
			}
		}
	}()
	return sizes
}
