// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package netutil provides utility functions for networking, including
// parsing IP addresses and prefixes.
package netutil

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// ParseAddrOrPrefix parses an IP address or prefix string and returns a
// netip.Prefix. If the string is an IP address without a prefix, it is treated
// as a full-bit prefix (/32 for IPv4, /128 for IPv6).
// ParseAddrOrPrefix calls ResolveFirst to resolve the address before parsing it.
func ParseAddrOrPrefix(addr string) (netip.Prefix, error) {
	// The "/prefixlen" suffix, if any, must be split off before resolving:
	// the host (or literal IP) is what gets resolved, not the whole string.
	host := addr
	prefixSuffix := ""
	if idx := strings.LastIndex(addr, "/"); idx >= 0 {
		host, prefixSuffix = addr[:idx], addr[idx:]
	}
	host, err := ResolveFirst(host)
	if err != nil {
		return netip.Prefix{}, err
	}
	if prefixSuffix == "" {
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return netip.Prefix{}, err
		}
		bits := 128
		if ip.Is4() {
			bits = 32
		}
		return netip.PrefixFrom(ip, bits), nil
	}
	return netip.ParsePrefix(host + prefixSuffix)
}

// ParseAddrIgnoringPort parses an IP address string and returns the address.
// If the string is an address with a port, it will be parsed as an address
// with a port and the address will be returned, ignoring the port.
// ParseAddrIgnoringPort calls ResolveFirst to resolve the address before parsing it.
func ParseAddrIgnoringPort(addr string) (netip.Addr, error) {
	addr, err := ResolveFirst(addr)
	if err != nil {
		return netip.Addr{}, err
	}
	ap, err := netip.ParseAddrPort(addr)
	if err == nil {
		return ap.Addr(), nil
	}
	return netip.ParseAddr(addr)
}

// ParseAddrDefaultPort parses an IP address string. If the address string
// already contains a port, it is parsed and returned. Otherwise, the
// supplied default port is used to construct and parse an address with
// that port. If the address contains only a port an address of "::" is
// used. ParseAddrDefaultPort calls ResolveFirst to resolve the address before
// parsing it.
func ParseAddrDefaultPort(addr, defaultPort string) (netip.AddrPort, error) {
	addr, err := ResolveFirst(addr)
	if err != nil {
		return netip.AddrPort{}, err
	}
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		// Addr is of the form :<port> with no address.
		if host == "" {
			host = "::"
		}
		return netip.ParseAddrPort(net.JoinHostPort(host, port))
	}
	if len(addr) == 0 {
		addr = "::"
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

// Resolve replaces the host component of addr with the first IP address
// resolved for that host. If the host component of addr cannot be resolved,
// addr is returned unchanged.
//
// Deprecated: Use ResolveFirst instead, which returns an error if the host
// cannot be resolved instead of returning the original address.
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

// ResolveAll returns all IP addresses resolved for the host component of addr.
func ResolveAll(addr string) ([]net.IP, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = TrimIPv6(host)
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	return net.LookupIP(host)
}

// ResolveFirst returns addr with the host component replaced by the first IP
// address resolved for that host; any port is preserved unchanged. If the
// host component is empty (e.g. "" or ":80") addr is returned unchanged since
// there is nothing to resolve. If the host cannot be resolved, an error is
// returned.
func ResolveFirst(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, ""
	}
	if host == "" {
		return addr, nil
	}
	ips, err := ResolveAll(host)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no addresses resolved for %q", host)
	}
	if port == "" {
		return ips[0].String(), nil
	}
	return net.JoinHostPort(ips[0].String(), port), nil
}

// EnsureHostPort returns addr that is guaranteed to have a port.
// If addr already has a port, it is returned unchanged. If addr does not have
// a port, the supplied port is appended. If addr is an IPv6 address,
// it will be enclosed in brackets if it is not already.
func EnsureHostPort(addr, port string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	// Avoid double brackets for IPv6 addresses.
	if len(addr) > 0 && addr[0] == '[' && addr[len(addr)-1] == ']' {
		addr = addr[1 : len(addr)-1]
	}
	return net.JoinHostPort(TrimIPv6(addr), port)
}

// TrimIPv6 removes brackets from an IPv6 address if they are present.
func TrimIPv6(addr string) string {
	if len(addr) > 0 && addr[0] == '[' && addr[len(addr)-1] == ']' {
		return addr[1 : len(addr)-1]
	}
	return addr
}
