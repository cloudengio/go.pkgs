// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build linux

package netutil

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// bindToInterface binds the socket with SO_BINDTODEVICE, which confines it to
// one interface and so to that interface's routes.
//
// Linux differs from the BSDs in that its route lookup does take a source
// address into account, so choosing one can select a path here where it
// cannot there. Binding the device is still the direct way to say which
// interface to use. It may be refused as a privileged operation, in which
// case the caller falls back to choosing a source address.
func bindToInterface(ifi net.Interface) func(network, address string, c syscall.RawConn) error {
	return func(_, _ string, c syscall.RawConn) error {
		var operr error
		if err := c.Control(func(fd uintptr) {
			operr = unix.BindToDevice(int(fd), ifi.Name)
		}); err != nil {
			return err
		}
		if operr != nil {
			return fmt.Errorf("binding a socket to %v: %w", ifi.Name, operr)
		}
		return nil
	}
}
