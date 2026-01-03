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
// address. If the string is an IP address without a prefix, it is treated
// as a full-bit prefix (/32 for IPv4, /128 for IPv6).
func ParseAddrOrPrefix(addr string) (netip.Addr, error) {
	if strings.Contains(addr, "/") {
		p, err := netip.ParsePrefix(addr)
		if err != nil {
			return netip.Addr{}, err
		}
		return p.Addr(), nil
	}
	return netip.ParseAddr(addr)
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

// ParseAddrDefaultPort parses an IP address string. If the address string
// already contains a port, it is parsed and returned. Otherwise, the
// supplied default port is used to construct and parse an address with
// that port. If the address contains only a port an address of "0.0.0.0"
// is used.
func ParseAddrDefaultPort(addr, defaultPort string) (netip.AddrPort, error) {
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		// Addr is of the form :<port> with no address.
		if host == "" {
			host = "0.0.0.0"
		}
		return netip.ParseAddrPort(net.JoinHostPort(host, port))
	}
	if len(addr) == 0 {
		addr = "0.0.0.0"
	}
	switch defaultPort {
	case "http":
		defaultPort = "80"
	case "https":
		defaultPort = "443"
	}
	return netip.ParseAddrPort(net.JoinHostPort(addr, defaultPort))
}

// HTTPServerAddr returns the address of an HTTP server based on the
// address and port of the server in a form to be used with http.Server.Addr.
// If the address is unspecified then it will be replaced with an empty string.
// If the port is 80 then "http" will be appended to the address, if the
// port is 443 then "https" will be appended to the address, otherwise the
// numeric port will be used.
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

// Resolve replaces the address component of addr with the first IP address
// resolved for the host component of addr. If the host component of addr
// cannot be resolved, addr is returned unchanged.
func Resolve(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return addr
	}
	if len(port) == 0 {
		return ips[0].String()
	}
	return net.JoinHostPort(ips[0].String(), port)
}
