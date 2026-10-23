// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package netutil

import (
	"context"
	"errors"
	"net"
	"syscall"
)

// Dialer is a net.Dialer that recovers from a host which is unreachable over
// the route the system chooses for it, but reachable over another, by
// retrying the connection from each local interface in turn.
//
// It exists for a host on a subnet that the system is attached to more than
// once, such as a machine on the same LAN over both wired ethernet and wi-fi.
// Each attachment contributes its own route to that subnet and a connection
// goes out of whichever of them the kernel selects. When it selects one whose
// neighbour discovery for the target has not succeeded, the connection fails
// at once with EHOSTUNREACH, "no route to host", even though the host is
// reachable over the other. Which route is selected is not stable, so the
// same address can be reachable from one process, or at one moment, and not
// from the next.
//
// Dial and DialContext first dial exactly as the embedded net.Dialer does.
// Only if that fails with EHOSTUNREACH do they try again, binding each
// attempt to a different local address, which selects the route by source
// address. The first connection to succeed is returned. If none does, the
// error from the original attempt is returned rather than the last one,
// since it describes the path the system was configured to use.
//
// The recovery is opt-in: RecoverNoRouteToHost must be set for any of it to
// happen. A host that cannot be reached is usually simply not reachable, and
// trying every interface then only delays the error and obscures it; the
// recovery is for a system attached to one subnet more than once, which the
// caller knows about and this type cannot detect. The zero value is a usable
// Dialer that dials exactly as its embedded net.Dialer does.
//
// A Dialer with its LocalAddr set is left alone even when the recovery is
// enabled: a caller that has chosen a source address has already said which
// path to use, so there is nothing to recover from.
//
// Timeout bounds the original attempt and then, separately, all of the
// retries together, so a dial takes at most about twice it rather than it
// once per local address. Deadline, being absolute, bounds everything as it
// would for a net.Dialer. Note that a route which cannot reach the host
// fails at once, without consuming either, which is why cycling through the
// interfaces of a multihomed system is cheap.
//
// Only EHOSTUNREACH is recovered from. It means the route that was used
// cannot reach the host, which is what another route may fix. A refused
// connection or a timeout says the route reached something, and trying from
// elsewhere would only obscure that.
type Dialer struct {
	net.Dialer

	// RecoverNoRouteToHost enables retrying a connection that failed with
	// EHOSTUNREACH by way of each local interface in turn, as described
	// above. It is off by default.
	RecoverNoRouteToHost bool
}

// Dial connects to the address on the named network, as per
// net.Dialer.Dial, recovering from EHOSTUNREACH as described for Dialer when
// RecoverNoRouteToHost is set.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network, as per
// net.Dialer.DialContext, recovering from EHOSTUNREACH as described for
// Dialer when RecoverNoRouteToHost is set.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err == nil || !isNoRouteToHost(err) {
		return conn, err
	}

	if !d.canRetryFrom(network) {
		return nil, err
	}
	// The source address the system would have used is the one whose route
	// has just been shown not to reach the host, so it is not worth using
	// again. Failing to determine it costs only a repeated attempt.
	if conn, ok := d.dialFromInterfaces(ctx, network, address, chosenSourceIP(ctx, network, address)); ok {
		return conn, nil
	}
	return nil, err
}

// canRetryFrom reports whether recovering a connection on this network is
// something this Dialer may do.
func (d *Dialer) canRetryFrom(network string) bool {
	return d.RecoverNoRouteToHost &&
		d.LocalAddr == nil &&
		localAddrForNetwork(network, net.IPv4zero) != nil
}

// dialFromInterfaces attempts to connect from each suitable local address in
// turn, skipping exclude, and reports whether one of them succeeded. The
// errors from the attempts are discarded: the caller has the error that
// describes the configured path, which is the more useful of them.
func (d *Dialer) dialFromInterfaces(ctx context.Context, network, address string, exclude net.IP) (net.Conn, bool) {
	// The attempts share one budget rather than each being allowed the whole
	// Timeout, so that recovering cannot multiply the time a caller asked a
	// dial to take by the number of local addresses. A Deadline needs no such
	// treatment: it is absolute, so it already bounds them all.
	if d.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.Timeout)
		defer cancel()
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, false
	}
	want4, want6 := targetFamilies(address)
	if conn, ok := d.dialBoundToInterface(ctx, network, address, ifaces, want4, want6); ok {
		return conn, true
	}
	return d.dialFromSourceAddr(ctx, network, address, ifaces, want4, want6, exclude)
}

// dialBoundToInterface attempts to connect with the socket bound to each
// interface that has an address of a family the target has.
func (d *Dialer) dialBoundToInterface(ctx context.Context, network, address string, ifaces []net.Interface, want4, want6 bool) (net.Conn, bool) {
	for _, ifc := range ifaces {
		if !hasUsableFamily(ifc, want4, want6) {
			continue
		}
		bind := bindToInterface(ifc)
		if bind == nil {
			return nil, false // This platform cannot bind an interface at all.
		}
		nd := d.Dialer
		bindSocket(&nd, bind)
		if conn, err := nd.DialContext(ctx, network, address); err == nil {
			return conn, true
		}
		if ctx.Err() != nil {
			return nil, false
		}
	}
	return nil, false
}

// dialFromSourceAddr attempts to connect from each suitable local address in
// turn, skipping exclude, which is the address the system has already been
// shown not to reach the host from.
func (d *Dialer) dialFromSourceAddr(ctx context.Context, network, address string, ifaces []net.Interface, want4, want6 bool, exclude net.IP) (net.Conn, bool) {
	for _, ifc := range ifaces {
		for _, ip := range usableAddrs(ifc) {
			if exclude != nil && ip.Equal(exclude) {
				continue
			}
			if !wantedFamily(ip, want4, want6) {
				continue
			}
			nd := d.Dialer
			nd.LocalAddr = localAddrForNetwork(network, ip)
			if conn, err := nd.DialContext(ctx, network, address); err == nil {
				return conn, true
			}
			if ctx.Err() != nil {
				return nil, false
			}
		}
	}
	return nil, false
}

// bindSocket adds bind to the dialer's socket controls, keeping any the
// caller set. A net.Dialer ignores Control when ControlContext is set, so a
// bind added to the one it ignores would silently not happen.
func bindSocket(d *net.Dialer, bind func(network, address string, c syscall.RawConn) error) {
	switch {
	case d.ControlContext != nil:
		caller := d.ControlContext
		d.ControlContext = func(ctx context.Context, network, address string, c syscall.RawConn) error {
			if err := caller(ctx, network, address, c); err != nil {
				return err
			}
			return bind(network, address, c)
		}
	case d.Control != nil:
		caller := d.Control
		d.Control = func(network, address string, c syscall.RawConn) error {
			if err := caller(network, address, c); err != nil {
				return err
			}
			return bind(network, address, c)
		}
	default:
		d.Control = bind
	}
}

// hasUsableFamily reports whether ifc has an address that a connection to a
// target of these families can be sourced from.
func hasUsableFamily(ifc net.Interface, want4, want6 bool) bool {
	for _, ip := range usableAddrs(ifc) {
		if wantedFamily(ip, want4, want6) {
			return true
		}
	}
	return false
}

// wantedFamily reports whether ip is of a family that the target has an
// address in.
func wantedFamily(ip net.IP, want4, want6 bool) bool {
	if ip.To4() != nil {
		return want4
	}
	return want6
}

// usableAddrs returns the addresses of ifc that a connection to another host
// can be sourced from. An interface that is down has no route, and loopback
// and link-local addresses cannot reach a host that the system's own routes
// could not.
func usableAddrs(ifc net.Interface) []net.IP {
	if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
		return nil
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return nil
	}
	var ips []net.IP
	for _, addr := range addrs {
		var ip net.IP
		switch a := addr.(type) {
		case *net.IPNet:
			ip = a.IP
		case *net.IPAddr:
			ip = a.IP
		default:
			continue
		}
		if ip == nil || ip.IsUnspecified() || ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

// targetFamilies reports which address families address resolves to, so that
// a source address of the wrong family is not bound for it. Both are reported
// when that cannot be determined, leaving the attempt to say.
func targetFamilies(address string) (ipv4, ipv6 bool) {
	ips, err := ResolveAll(address)
	if err != nil || len(ips) == 0 {
		return true, true
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = true
			continue
		}
		ipv6 = true
	}
	return ipv4, ipv6
}

// chosenSourceIP returns the local address the system would use to reach
// address, or nil if that cannot be determined. A connected udp socket
// selects a route and binds a local address to it without sending anything,
// which is what reveals the choice.
func chosenSourceIP(ctx context.Context, network, address string) net.IP {
	probeNetwork, ok := probeNetworkFor(network)
	if !ok {
		return nil
	}
	var probe net.Dialer
	conn, err := probe.DialContext(ctx, probeNetwork, address)
	if err != nil {
		return nil
	}
	defer conn.Close()
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil
	}
	return local.IP
}

// probeNetworkFor returns the connectionless network of the same address
// family as network, which chosenSourceIP uses to observe a route choice
// without making a connection.
func probeNetworkFor(network string) (string, bool) {
	switch network {
	case "tcp", "udp":
		return "udp", true
	case "tcp4", "udp4":
		return "udp4", true
	case "tcp6", "udp6":
		return "udp6", true
	}
	return "", false
}

// localAddrForNetwork returns ip as the local address of a connection on the
// named network, with no port so that one is assigned, or nil for a network
// whose local address is not an ip address.
func localAddrForNetwork(network string, ip net.IP) net.Addr {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip}
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip}
	}
	return nil
}

// isNoRouteToHost reports whether err is the system saying that the route it
// used cannot reach the host.
func isNoRouteToHost(err error) bool {
	return errors.Is(err, syscall.EHOSTUNREACH)
}
