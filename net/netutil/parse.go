// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package netutil provides utility functions for networking, including
// parsing IP addresses and prefixes.
package netutil

import (
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// ParseAddrOrPrefix parses an IP address or prefix string and returns the
// address. If the string is a prefix, it will be parsed as a prefix and the
// address will be returned.
func ParseAddrOrPrefix(addr string) (netip.Addr, error) {
	if !strings.Contains(addr, "/") {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return netip.Addr{}, err
		}
		if ip.Is4() {
			addr += "/32"
		} else {
			addr += "/128"
		}
	}
	p, err := netip.ParsePrefix(addr)
	if err != nil {
		return netip.Addr{}, err
	}
	return p.Addr(), nil
}

// ParseAddrIgnoringPort parses an IP address string and returns the address.
// If the string is an address with a port, it will be parsed as an address
// with a port and the address will be returned, ignoring the port.
func ParseAddrIgnoringPort(addr string) (netip.Addr, error) {
	ap, err := netip.ParseAddrPort(addr)
	if err == nil {
		return ap.Addr(), nil
	}
	return netip.ParseAddr(addr)
}

// ParseAddrDefaultPort parses an IP address with an optional port, if
// the port is specified the address it will be parsed as an address
// with a port and the address will be returned, otherwise the default
// port will be appended to the address and the address will be parsed
// as an address with that default port.
func ParseAddrDefaultPort(addr, port string) (netip.AddrPort, error) {
	ap, err := netip.ParseAddrPort(addr)
	if err == nil {
		return ap, nil
	}
	if len(addr) == 0 {
		addr = "0.0.0.0"
	}
	switch port {
	case "http":
		port = "80"
	case "https":
		port = "443"
	}
	return netip.ParseAddrPort(net.JoinHostPort(addr, port))
}

// HTTPServerAddr returns the address of an HTTP server based on the
// address and port of the server in a form to be used with http.Server.Addr.
// If the address is unspecified then it will be replaced with an empty string.
// If the port is 80 then "http" will be appended to the address, if the
// port is 443 then "https" will be appended to the address, otherwise the
// address will be returned as is.
func HTTPServerAddr(addrPort netip.AddrPort) string {
	var addr string
	if addrPort.Addr().IsUnspecified() {
		addr = ""
	} else {
		addr = addrPort.Addr().String()
	}
	switch addrPort.Port() {
	case 80:
		return net.JoinHostPort(addr, "http")
	case 443:
		return net.JoinHostPort(addr, "https")
	default:
		return net.JoinHostPort(addr, strconv.Itoa(int(addrPort.Port())))
	}
}
