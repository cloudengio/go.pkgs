// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"fmt"
	"net"

	"cloudeng.io/net/netutil"
)

// WithDialer replaces the dialer used to reach the server, so that the ssh
// connection can be carried by something other than a direct connection from
// this host, such as another ssh connection (see WithProxyHost) or a SOCKS
// proxy. Only the transport changes: the server is still verified by its own
// host key, under the address that NewClient was given.
//
// The dialer is called once per attempt to establish the connection, and the
// context it is given bounds that attempt alone, as it does for
// net.Dialer.DialContext: the connection it returns must not be closed when
// that context is done.
//
// It replaces the dialer this package would otherwise use, so
// WithInterfaceRecovery no longer has anything to apply to. WithDialTimeout
// still applies, as the deadline of the context the dialer is given.
func WithDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(o *options) {
		o.dialer = dialer
	}
}

// WithProxyHost reaches the server over the connection that proxy maintains,
// rather than by connecting to it directly, as ssh's ProxyJump ('ssh -J')
// does, for a server behind a bastion or in a private network: each attempt
// asks proxy to open a connection to this client's address, so the server need
// not be reachable from this host at all. The server is verified by its own
// host key under its own name, since only the transport changes.
//
// proxy is an ordinary client, which the caller establishes and maintains with
// its own ConnectAndWait; an attempt to connect to the server waits for it to
// be connected, bounded by WithDialTimeout. The two then recover together: the
// connection to the server is carried by proxy's connection, so losing that
// loses this one too, and this one is established again once proxy has
// reconnected.
//
// The caller must Close proxy once its ConnectAndWait has returned, which is
// what ends this client as well: an attempt through a closed proxy host fails
// with ErrClientClosed, which is not retryable and so is returned by
// ConnectAndWait. Without that close a client would retry indefinitely
// through a proxy host that has itself given up.
func WithProxyHost(proxy *Client) Option {
	return func(o *options) {
		o.proxyAddr = proxy.addr
		o.dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := proxy.DialContext(ctx, network, addr)
			if err != nil {
				// Named because the failure is the proxy host's rather than
				// the server's, which is otherwise indistinguishable here.
				return nil, fmt.Errorf("via proxy host %v: %w", proxy.addr, err)
			}
			return conn, nil
		}
	}
}

// DialContext connects to addr from the server, over the connection to it, as
// the far end of a local port forward does. It waits for the client to be
// connected, as Run does, returning ctx.Err() if ctx is done first and
// ErrClientClosed if the client is closed.
//
// The connection it returns is carried by the connection to the server and so
// does not outlive it: when that is lost this one is closed too.
func (c *Client) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	for {
		conn, err := c.connection(ctx)
		if err != nil {
			return nil, err
		}
		tunnelled, err := conn.DialContext(ctx, network, addr)
		if err == nil {
			return tunnelled, nil
		}
		if !isConnLost(err) {
			return nil, err
		}
		// The connection to the server has been lost, rather than addr having
		// refused this one, so wait for its replacement and ask that instead,
		// as newSession does.
		c.clearConn(conn)
	}
}

// dial makes the connection that the ssh handshake runs over: the one supplied
// by WithDialer or WithProxyHost, or a direct connection to the server.
func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	if c.opts.dialer == nil {
		// netutil.Dialer dials identically to a net.Dialer unless the recovery
		// is asked for, in which case a host on a subnet this system is
		// attached to more than once is still reached when the route the
		// kernel picks for it cannot: see WithInterfaceRecovery and
		// netutil.Dialer.
		dialer := netutil.Dialer{
			Dialer:               net.Dialer{Timeout: c.opts.dialTimeout},
			RecoverNoRouteToHost: c.opts.recoverNoRoute,
		}
		return dialer.DialContext(ctx, c.network, c.addr)
	}
	// A supplied dialer has no timeout of its own, as net.Dialer does, so the
	// context is what bounds an attempt that would otherwise hang: a proxy host
	// that is not connected, or one that is but whose server never answers the
	// request to open the connection.
	dialCtx, cancel := context.WithTimeout(ctx, c.opts.dialTimeout)
	defer cancel()
	return c.opts.dialer(dialCtx, c.network, c.addr)
}
