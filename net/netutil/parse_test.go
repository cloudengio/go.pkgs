// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package netutil_test

import (
	"net/netip"
	"slices"
	"testing"

	"cloudeng.io/net/netutil"
)

func TestParseAddrOrPrefix(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"192.168.1.1", "192.168.1.1"},
		{"192.168.1.1/32", "192.168.1.1"},
		{"192.168.1.1/24", "192.168.1.1"},
		{"::1", "::1"},
		{"::1/128", "::1"},
	} {
		addr, err := netutil.ParseAddrOrPrefix(tc.input)
		if err != nil {
			t.Errorf("ParseAddrOrPrefix(%q): %v", tc.input, err)
			continue
		}
		if got := addr.String(); got != tc.want {
			t.Errorf("ParseAddrOrPrefix(%q): got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseAddrIgnoringPort(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"192.168.1.1", "192.168.1.1"},
		{"192.168.1.1:80", "192.168.1.1"},
		{"[::1]:80", "::1"},
		{"::1", "::1"},
	} {
		addr, err := netutil.ParseAddrIgnoringPort(tc.input)
		if err != nil {
			t.Errorf("ParseAddrIgnoringPort(%q): %v", tc.input, err)
			continue
		}
		if got := addr.String(); got != tc.want {
			t.Errorf("ParseAddrIgnoringPort(%q): got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseAddrDefaultPort(t *testing.T) {
	for _, tc := range []struct {
		input       string
		defaultPort string
		want        string
	}{
		{"192.168.1.1:80", "443", "192.168.1.1:80"},
		{"192.168.1.1", "80", "192.168.1.1:80"},
		{"192.168.1.1", "http", "192.168.1.1:80"},
		{"192.168.1.1", "https", "192.168.1.1:443"},
		{"::1", "80", "[::1]:80"},
		{"", "80", "[::]:80"},
		{"", "http", "[::]:80"},
		{"", "https", "[::]:443"},
		{":80", "443", "[::]:80"},
	} {
		addr, err := netutil.ParseAddrDefaultPort(tc.input, tc.defaultPort)
		if err != nil {
			t.Errorf("ParseAddrDefaultPort(%q, %q): %v", tc.input, tc.defaultPort, err)
			continue
		}
		if got := addr.String(); got != tc.want {
			t.Errorf("ParseAddrDefaultPort(%q, %q): got %v, want %v", tc.input, tc.defaultPort, got, tc.want)
		}
	}
}

func TestHTTPServerAddr(t *testing.T) {
	for _, tc := range []struct {
		addr netip.AddrPort
		want string
	}{
		{netip.AddrPortFrom(netip.MustParseAddr("0.0.0.0"), 80), ":http"},
		{netip.AddrPortFrom(netip.MustParseAddr("0.0.0.0"), 443), ":https"},
		{netip.AddrPortFrom(netip.MustParseAddr("1.1.1.1"), 80), "1.1.1.1:http"},
		{netip.AddrPortFrom(netip.MustParseAddr("1.1.1.1"), 443), "1.1.1.1:https"},
		{netip.AddrPortFrom(netip.MustParseAddr("1.1.1.1"), 8080), "1.1.1.1:8080"},
	} {
		if got := netutil.HTTPServerAddr(tc.addr); got != tc.want {
			t.Errorf("HTTPServerAddr(%v): got %v, want %v", tc.addr, got, tc.want)
		}
	}

}

func TestResolveInFunctions(t *testing.T) {
	// Test ParseAddrOrPrefix with Resolve
	addr, err := netutil.ParseAddrOrPrefix("localhost")
	if err != nil {
		t.Errorf("ParseAddrOrPrefix(\"localhost\"): %v", err)
	} else if got := addr.String(); got != "127.0.0.1" && got != "::1" {
		t.Errorf("ParseAddrOrPrefix(\"localhost\"): got %v, want 127.0.0.1 or ::1", got)
	}

	// Test ParseAddrIgnoringPort with Resolve
	addr, err = netutil.ParseAddrIgnoringPort("localhost:80")
	if err != nil {
		t.Errorf("ParseAddrIgnoringPort(\"localhost:80\"): %v", err)
	} else if got := addr.String(); got != "127.0.0.1" && got != "::1" {
		t.Errorf("ParseAddrIgnoringPort(\"localhost:80\"): got %v, want 127.0.0.1 or ::1", got)
	}

	// Test ParseAddrDefaultPort with Resolve
	ap, err := netutil.ParseAddrDefaultPort("localhost", "80")
	if err != nil {
		t.Errorf("ParseAddrDefaultPort(\"localhost\", \"80\"): %v", err)
	} else if got := ap.String(); got != "127.0.0.1:80" && got != "[::1]:80" {
		t.Errorf("ParseAddrDefaultPort(\"localhost\", \"80\"): got %v, want 127.0.0.1:80 or [::1]:80", got)
	}
}

func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []string
	}{
		{"localhost:80", []string{"127.0.0.1:80", "[::1]:80"}},
		{"localhost", []string{"127.0.0.1", "::1"}},
		{"localhost:http", []string{"127.0.0.1:http", "[::1]:http"}},
		{"localhost:https", []string{"127.0.0.1:https", "[::1]:https"}},
		{"127.0.0.1:80", []string{"127.0.0.1:80"}},
		{"[::1]:80", []string{"[::1]:80"}},
		{"host.invalid:80", []string{"host.invalid:80"}},
		{"host.invalid", []string{"host.invalid"}},
		{"", []string{""}},
		{":80", []string{":80"}},
	} {
		got := netutil.Resolve(tc.input)
		found := slices.Contains(tc.want, got)
		if !found {
			t.Errorf("Resolve(%q): got %v, want one of %v", tc.input, got, tc.want)
		}
	}
}
