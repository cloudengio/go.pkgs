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
	"runtime"
	"slices"
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
	// The recovery is opt-in: a host that cannot be reached is usually not
	// reachable, and trying every interface then only delays the error.
	var off Dialer
	for _, network := range []string{"tcp", "tcp4", "tcp6", "udp"} {
		if off.canRetryFrom(network) {
			t.Errorf("%v: the zero value should dial as a net.Dialer does", network)
		}
	}

	on := Dialer{RecoverNoRouteToHost: true}
	for _, network := range []string{"tcp", "tcp4", "tcp6", "udp"} {
		if !on.canRetryFrom(network) {
			t.Errorf("%v: should recover once asked to", network)
		}
	}
	for _, network := range []string{"unix", "unixgram", "ip:icmp", ""} {
		if on.canRetryFrom(network) {
			t.Errorf("%v: there is no source address to choose", network)
		}
	}

	// A caller that chose a source address has already said which path to
	// use, so it is left alone even when the recovery is asked for.
	chosen := Dialer{
		Dialer:               net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP("10.1.2.3")}},
		RecoverNoRouteToHost: true,
	}
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

	// Listening on every interface so that the connection can be made by way
	// of any of them, and dialling a non-loopback address of this machine so
	// that the recovery path, which skips loopback, can reach it.
	ln, port := listenOnEveryInterface(t)
	defer ln.Close()
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

	// Binding the interface is tried for every interface, including the one
	// the system would have chosen: a socket bound to an interface and one
	// left to the system do not take the same path through the routing table,
	// so excluding it would skip an attempt that can succeed where the
	// unbound one did not. The exclusion applies to the source address
	// fallback, which TestDialFromSourceAddrExcludes covers.
	conn, ok = d.dialFromInterfaces(t.Context(), "tcp", target, local.IP)
	if !ok {
		t.Fatalf("no interface could reach %v", target)
	}
	conn.Close()
}

// targetsPerFamily returns an address of the listener for one usable local
// address of each family this machine has, so that both are exercised
// wherever both exist. Which family comes first differs between machines,
// which is how a hardcoded family in these tests passed here and failed on a
// macos runner whose first usable address is an ipv6 one.
func targetsPerFamily(t *testing.T, port int) []string {
	t.Helper()
	var targets []string
	var seen4, seen6 bool
	for _, ip := range globalUnicastAddrs(t) {
		if is4 := ip.To4() != nil; is4 && !seen4 {
			seen4 = true
		} else if !is4 && !seen6 {
			seen6 = true
		} else {
			continue
		}
		targets = append(targets, net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port)))
	}
	return targets
}

func TestDialBoundToInterface(t *testing.T) {
	addrs := globalUnicastAddrs(t)
	if len(addrs) == 0 {
		t.Skip("no interface has an address a connection can be sourced from")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("net.Interfaces: %v", err)
	}
	if bindToInterface(ifaces[0]) == nil {
		t.Skipf("%v cannot bind a socket to an interface", runtime.GOOS)
	}

	ln, port := listenOnEveryInterface(t)
	defer ln.Close()

	for _, target := range targetsPerFamily(t, port) {
		t.Run(target, func(t *testing.T) {
			// A control the caller set must still be called: the bind is
			// added to it rather than replacing it.
			var controlled int
			d := Dialer{Dialer: net.Dialer{
				Control: func(string, string, syscall.RawConn) error {
					controlled++
					return nil
				},
			}}
			want4, want6 := targetFamilies(target)
			conn, ok := d.dialBoundToInterface(t.Context(), "tcp", target, ifaces, want4, want6)
			if !ok {
				t.Fatalf("no interface could reach %v", target)
			}
			defer conn.Close()
			if controlled == 0 {
				t.Error("the caller's Control was not called")
			}
			if !containsIP(addrs, conn.LocalAddr().(*net.TCPAddr).IP) {
				t.Errorf("connected from %v, which is not an interface address", conn.LocalAddr())
			}
		})
	}
}

func TestDialFromSourceAddrExcludes(t *testing.T) {
	addrs := globalUnicastAddrs(t)
	if len(addrs) == 0 {
		t.Skip("no interface has an address a connection can be sourced from")
	}
	ln, port := listenOnEveryInterface(t)
	defer ln.Close()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("net.Interfaces: %v", err)
	}

	for _, target := range targetsPerFamily(t, port) {
		t.Run(target, func(t *testing.T) {
			want4, want6 := targetFamilies(target)
			var d Dialer
			conn, ok := d.dialFromSourceAddr(t.Context(), "tcp", target, ifaces, want4, want6, nil)
			if !ok {
				t.Fatalf("no address could reach %v", target)
			}
			used := conn.LocalAddr().(*net.TCPAddr).IP
			conn.Close()

			// The excluded address is the one the system has already been
			// shown not to reach the host from, so it is not worth binding
			// again.
			conn, ok = d.dialFromSourceAddr(t.Context(), "tcp", target, ifaces, want4, want6, used)
			if !ok {
				return // No other address can reach it, which is not a failure.
			}
			defer conn.Close()
			if got := conn.LocalAddr().(*net.TCPAddr).IP; got.Equal(used) {
				t.Errorf("connected from %v, which was excluded", got)
			}
		})
	}
}

func TestBindSocket(t *testing.T) {
	var calls []string
	bind := func(string, string, syscall.RawConn) error {
		calls = append(calls, "bind")
		return nil
	}

	// With no control of its own, the bind becomes one.
	var plain net.Dialer
	bindSocket(&plain, bind)
	if plain.Control == nil {
		t.Fatal("no Control was set")
	}
	if err := plain.Control("tcp4", "a", nil); err != nil {
		t.Errorf("Control: %v", err)
	}

	// With a Control, both are called and the caller's goes first.
	calls = nil
	withControl := net.Dialer{Control: func(string, string, syscall.RawConn) error {
		calls = append(calls, "caller")
		return nil
	}}
	bindSocket(&withControl, bind)
	if err := withControl.Control("tcp4", "a", nil); err != nil {
		t.Errorf("Control: %v", err)
	}
	if !slices.Equal(calls, []string{"caller", "bind"}) {
		t.Errorf("got %v, want the caller's control and then the bind", calls)
	}

	// A net.Dialer ignores Control when ControlContext is set, so the bind
	// has to be added to that one or it would never happen.
	calls = nil
	withContext := net.Dialer{ControlContext: func(context.Context, string, string, syscall.RawConn) error {
		calls = append(calls, "caller")
		return nil
	}}
	bindSocket(&withContext, bind)
	if withContext.Control != nil {
		t.Error("Control was set, which a dialer with a ControlContext ignores")
	}
	if err := withContext.ControlContext(t.Context(), "tcp4", "a", nil); err != nil {
		t.Errorf("ControlContext: %v", err)
	}
	if !slices.Equal(calls, []string{"caller", "bind"}) {
		t.Errorf("got %v, want the caller's control and then the bind", calls)
	}

	// A control that fails stops the socket being bound, and says so.
	calls = nil
	failing := net.Dialer{Control: func(string, string, syscall.RawConn) error {
		return errors.New("refused")
	}}
	bindSocket(&failing, bind)
	if err := failing.Control("tcp4", "a", nil); err == nil || err.Error() != "refused" {
		t.Errorf("got %v, want the caller's error", err)
	}
	if len(calls) != 0 {
		t.Errorf("the socket was bound after the caller's control failed: %v", calls)
	}
}

// listenOnEveryInterface returns a listener reachable on any interface, so
// that a connection made by way of any of them arrives, and its port.
func listenOnEveryInterface(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := net.Listen("tcp", ":0") //nolint:gosec // G102: the recovery path skips loopback, so it has to be reachable on a real interface.
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return ln, ln.Addr().(*net.TCPAddr).Port
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
	d := Dialer{Dialer: net.Dialer{Timeout: 250 * time.Millisecond}}
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
