// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// This file tests the recovery path of Dialer, which cannot be reached from
// outside the package: it requires the system to produce EHOSTUNREACH, which
// a test cannot arrange.
package netutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

// hostUnreachable is the error the system returns for a route that cannot
// reach the host, as net wraps it.
func hostUnreachable() error {
	return &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: os.NewSyscallError("connect", syscall.EHOSTUNREACH),
	}
}

func TestIsNoRouteToHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"no route to host", hostUnreachable(), true},
		{"wrapped again", fmt.Errorf("dialling: %w", hostUnreachable()), true},
		{"bare errno", syscall.EHOSTUNREACH, true},
		// A refused connection or a timeout means the route reached
		// something, so another route is not the answer.
		{"connection refused", &net.OpError{Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}, false},
		{"network unreachable", &net.OpError{Err: os.NewSyscallError("connect", syscall.ENETUNREACH)}, false},
		{"timeout", os.ErrDeadlineExceeded, false},
		{"unrelated", errors.New("no route to host"), false},
		{"nil", nil, false},
	} {
		if got := isNoRouteToHost(tc.err); got != tc.want {
			t.Errorf("%v: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLocalAddrForNetwork(t *testing.T) {
	ip := net.ParseIP("10.1.2.3")
	for _, tc := range []struct {
		network string
		want    net.Addr
	}{
		{"tcp", &net.TCPAddr{IP: ip}},
		{"tcp4", &net.TCPAddr{IP: ip}},
		{"tcp6", &net.TCPAddr{IP: ip}},
		{"udp", &net.UDPAddr{IP: ip}},
		{"udp4", &net.UDPAddr{IP: ip}},
		{"udp6", &net.UDPAddr{IP: ip}},
		// A network whose local address is not an ip address has no source
		// address to choose, so there is nothing to recover from.
		{"unix", nil},
		{"ip:icmp", nil},
		{"", nil},
	} {
		got := localAddrForNetwork(tc.network, ip)
		if (got == nil) != (tc.want == nil) {
			t.Errorf("%v: got %v, want %v", tc.network, got, tc.want)
			continue
		}
		if got == nil {
			continue
		}
		if got.Network() != tc.want.Network() || got.String() != tc.want.String() {
			t.Errorf("%v: got %v (%v), want %v (%v)", tc.network,
				got, got.Network(), tc.want, tc.want.Network())
		}
		// No port is requested, so one is assigned.
		if want := ip.String() + ":0"; got.String() != want {
			t.Errorf("%v: got %v, want %v", tc.network, got, want)
		}
	}
}

func TestProbeNetworkFor(t *testing.T) {
	for _, tc := range []struct {
		network, want string
		ok            bool
	}{
		// The probe must be of the same address family, or it would observe
		// a route choice that is not the one being recovered from.
		{"tcp", "udp", true},
		{"tcp4", "udp4", true},
		{"tcp6", "udp6", true},
		{"udp", "udp", true},
		{"udp4", "udp4", true},
		{"udp6", "udp6", true},
		{"unix", "", false},
		{"", "", false},
	} {
		got, ok := probeNetworkFor(tc.network)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%v: got %v, %v, want %v, %v", tc.network, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCanRetryFrom(t *testing.T) {
	var d Dialer
	for _, network := range []string{"tcp", "tcp4", "tcp6", "udp"} {
		if !d.canRetryFrom(network) {
			t.Errorf("%v: the zero value should recover", network)
		}
	}
	for _, network := range []string{"unix", "unixgram", "ip:icmp", ""} {
		if d.canRetryFrom(network) {
			t.Errorf("%v: there is no source address to choose", network)
		}
	}

	// A caller that chose a source address has already said which path to
	// use, so it is left alone.
	chosen := Dialer{net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP("10.1.2.3")}}}
	if chosen.canRetryFrom("tcp") {
		t.Error("an explicit LocalAddr should not be overridden")
	}
}

func TestTargetFamilies(t *testing.T) {
	for _, tc := range []struct {
		address      string
		want4, want6 bool
	}{
		{"127.0.0.1:80", true, false},
		{"[::1]:80", false, true},
		{"10.1.2.3:22", true, false},
		// An address that cannot be resolved leaves the attempt to say,
		// rather than ruling every source address out.
		{"no-such-host.invalid:80", true, true},
	} {
		got4, got6 := targetFamilies(tc.address)
		if got4 != tc.want4 || got6 != tc.want6 {
			t.Errorf("%v: got v4=%v v6=%v, want v4=%v v6=%v", tc.address, got4, got6, tc.want4, tc.want6)
		}
	}
}

func TestChosenSourceIP(t *testing.T) {
	// A connected udp socket binds the source address of the route to the
	// target without sending anything, which is the choice being reported.
	ip := chosenSourceIP(t.Context(), "tcp", "127.0.0.1:9")
	if ip == nil {
		t.Skip("the route to loopback could not be observed")
	}
	if !ip.IsLoopback() {
		t.Errorf("got %v, want a loopback address for a loopback target", ip)
	}

	// A network with no source address to choose has no choice to report.
	if ip := chosenSourceIP(t.Context(), "unix", "/no/such/socket"); ip != nil {
		t.Errorf("got %v, want nil", ip)
	}
}

// globalUnicastAddrs returns every address the recovery path would try,
// across all interfaces.
func globalUnicastAddrs(t *testing.T) []net.IP {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("net.Interfaces: %v", err)
	}
	var ips []net.IP
	for _, ifc := range ifaces {
		ips = append(ips, usableAddrs(ifc)...)
	}
	return ips
}

func TestUsableAddrs(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("net.Interfaces: %v", err)
	}
	for _, ifc := range ifaces {
		addrs := usableAddrs(ifc)
		// An interface that is down has no route, and a connection to
		// another host cannot be sourced from loopback.
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			if len(addrs) != 0 {
				t.Errorf("%v: got %v, want none for flags %v", ifc.Name, addrs, ifc.Flags)
			}
			continue
		}
		for _, ip := range addrs {
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
				t.Errorf("%v: %v cannot reach a host the system's own routes could not", ifc.Name, ip)
			}
		}
	}
}

func TestDialFromInterfaces(t *testing.T) {
	addrs := globalUnicastAddrs(t)
	if len(addrs) == 0 {
		t.Skip("no interface has an address a connection can be sourced from")
	}

	// Listening on every interface so that the connection can be made from
	// any of them, and dialling a non-loopback address of this machine so
	// that the recovery path, which skips loopback, can reach it.
	ln, err := net.Listen("tcp", ":0") //nolint:gosec // G102: the recovery path skips loopback, so it has to be reachable on a real interface.
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	target := net.JoinHostPort(addrs[0].String(), fmt.Sprintf("%d", port))

	var d Dialer
	conn, ok := d.dialFromInterfaces(t.Context(), "tcp", target, nil)
	if !ok {
		t.Fatalf("no interface could reach %v", target)
	}
	// The connection is bound to one of the interface addresses rather than
	// left to the system, which is the whole point of the recovery.
	local := conn.LocalAddr().(*net.TCPAddr)
	conn.Close()
	if !containsIP(addrs, local.IP) {
		t.Errorf("connected from %v, which is not one of %v", local.IP, addrs)
	}

	// The excluded address is the one already shown not to work, so it must
	// not be used again, whether or not another one succeeds.
	conn, ok = d.dialFromInterfaces(t.Context(), "tcp", target, local.IP)
	if !ok {
		return // No other interface can reach it, which is not a failure.
	}
	defer conn.Close()
	if excluded := conn.LocalAddr().(*net.TCPAddr).IP; excluded.Equal(local.IP) {
		t.Errorf("connected from %v, which was excluded", excluded)
	}
}

func TestDialFromInterfacesCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// Nothing is listening, so every attempt fails, and a cancelled context
	// must stop them rather than work through every interface.
	var d Dialer
	if conn, ok := d.dialFromInterfaces(ctx, "tcp", "10.255.255.1:9", nil); ok {
		conn.Close()
		t.Error("a cancelled dial should not connect")
	}
}

func containsIP(ips []net.IP, want net.IP) bool {
	for _, ip := range ips {
		if ip.Equal(want) {
			return true
		}
	}
	return false
}

func TestDialFromInterfacesSharedBudget(t *testing.T) {
	if len(globalUnicastAddrs(t)) == 0 {
		t.Skip("no interface has an address a connection can be sourced from")
	}

	// An address reserved for documentation, so it is either unroutable or
	// routed nowhere. Whichever it is, every local address is tried, and the
	// attempts must share the budget rather than each being allowed it: the
	// point of the bound is that a dial does not take Timeout once per
	// interface.
	d := Dialer{net.Dialer{Timeout: 250 * time.Millisecond}}
	start := time.Now()
	conn, ok := d.dialFromInterfaces(t.Context(), "tcp", "192.0.2.1:9", nil)
	if ok {
		conn.Close()
		t.Skip("192.0.2.1 is reachable on this network")
	}
	// Generous enough not to be flaky, tight enough to catch the budget
	// being applied per address on a machine with several of them.
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v for a 250ms budget", elapsed)
	}
}
