// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package netutil

import (
	"fmt"
	"net"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// isIPv6Network reports whether a network, as the socket controls receive it
// and so already resolved to a family, is ipv6.
func isIPv6Network(network string) bool {
	return strings.HasSuffix(network, "6")
}

// bindToInterface binds the socket with IP_BOUND_IF, or IPV6_BOUND_IF for an
// ipv6 connection.
//
// This, and not the source address, is what selects the interface a
// connection leaves by. macOS scopes the routes of every interface other
// than the primary network service, and a scoped route is used only by a
// socket bound to its interface: the route lookup takes a destination and an
// interface scope and has no source address in it at all. It is also what
// connectx, which the system's own tools use, does on their behalf, which is
// why ssh and nc can reach a host that a plain connect cannot.
func bindToInterface(ifi net.Interface) func(network, address string, c syscall.RawConn) error {
	return func(network, _ string, c syscall.RawConn) error {
		level, option := unix.IPPROTO_IP, unix.IP_BOUND_IF
		if isIPv6Network(network) {
			level, option = unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF
		}
		var operr error
		if err := c.Control(func(fd uintptr) {
			operr = unix.SetsockoptInt(int(fd), level, option, ifi.Index)
		}); err != nil {
			return err
		}
		if operr != nil {
			return fmt.Errorf("binding a socket to %v: %w", ifi.Name, operr)
		}
		return nil
	}
}
