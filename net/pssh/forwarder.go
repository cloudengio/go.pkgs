// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"cloudeng.io/sync/ctxsync"
	"golang.org/x/crypto/ssh"
)

// forwarder runs the local port forwards for one ssh connection. Everything it
// starts is tracked by its WaitGroup and torn down by stopAll, so that a
// connection which is being replaced leaves nothing behind.
type forwarder struct {
	client   *ssh.Client
	forwards []forward
	logger   *slog.Logger
	wg       ctxsync.WaitGroup

	mu        sync.Mutex
	listeners []net.Listener
	conns     []net.Conn
	stopped   bool
}

func (f *forwarder) log() *slog.Logger {
	if f.logger == nil {
		return slog.Default()
	}
	return f.logger
}

// forwardAll establishes every configured forward. Each local port is bound
// before returning, so that a port which cannot be bound is reported here
// rather than asynchronously, whilst the connections accepted on it are
// handled by a goroutine tracked by the WaitGroup. A failure to bind leaves
// the forwards already established running: the caller is expected to call
// stopAll, which is what releases them.
func (f *forwarder) forwardAll(ctx context.Context) error {
	for _, spec := range f.forwards {
		l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", spec.localPort))
		if err != nil {
			return err
		}
		f.mu.Lock()
		f.listeners = append(f.listeners, l)
		f.mu.Unlock()
		f.wg.Go(func() {
			f.forward(ctx, l, spec)
		})
	}
	return nil
}

// forward accepts connections on l and forwards each of them over the ssh
// connection, returning when l is closed by stopAll. A connection that cannot
// be forwarded is abandoned rather than ending the forward, leaving the local
// port bound as ssh(1) does: the remote port may be listening again by the
// time the next connection arrives, and if it is the ssh connection that has
// failed then stopAll ends this loop in any case.
func (f *forwarder) forward(ctx context.Context, l net.Listener, spec forward) {
	remoteAddr := fmt.Sprintf("localhost:%d", spec.remotePort)
	for {
		local, err := l.Accept()
		if err != nil {
			// Either stopAll closed the listener or it can no longer be
			// used; both mean this forward is over.
			return
		}
		remote, err := f.client.DialContext(ctx, "tcp", remoteAddr)
		if err != nil {
			f.log().Warn("pssh: could not forward connection",
				"local.port", spec.localPort, "remote.port", spec.remotePort, "err", err)
			local.Close()
			continue
		}
		if !f.addConns(local, remote) {
			// stopAll ran between the accept and here, so these two
			// connections would not otherwise be closed by it.
			local.Close()
			remote.Close()
			return
		}
		f.wg.Go(func() {
			f.runForward(local, remote)
		})
	}
}

// addConns records local and remote for stopAll to close, reporting false if
// the forwarder has already been stopped and the caller must close them itself.
func (f *forwarder) addConns(local, remote net.Conn) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped {
		return false
	}
	f.conns = append(f.conns, local, remote)
	return true
}

func (f *forwarder) runForward(local, remote net.Conn) {
	// Copy both ways; closing either end must tear down the other. The copies
	// end with an error whenever the connection is torn down rather than shut
	// down cleanly, which is the usual case here and not worth reporting.
	f.wg.Go(func() {
		_, _ = io.Copy(remote, local)
		remote.Close()
	})
	_, _ = io.Copy(local, remote)
	local.Close()
}

// stopAll ends every forward and waits for the goroutines running them to
// finish. Both the listeners and the connections have to be closed for that
// wait to return: closing the listeners ends the accept loops, and closing the
// connections unblocks the copies in progress. It is safe to call more than
// once, and on a forwarder whose forwardAll failed part way through.
func (f *forwarder) stopAll(ctx context.Context) {
	f.mu.Lock()
	f.stopped = true
	for _, l := range f.listeners {
		l.Close()
	}
	f.listeners = nil
	for _, conn := range f.conns {
		conn.Close()
	}
	f.conns = nil
	f.mu.Unlock()
	f.wg.Wait(ctx)
}
