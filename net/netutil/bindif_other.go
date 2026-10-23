// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !darwin && !linux

package netutil

import (
	"net"
	"syscall"
)

// bindToInterface reports that this platform has no way to bind a socket to
// an interface, by returning nil. Choosing a source address is then all that
// is left, which is what the caller falls back to.
func bindToInterface(_ net.Interface) func(network, address string, c syscall.RawConn) error {
	return nil
}
