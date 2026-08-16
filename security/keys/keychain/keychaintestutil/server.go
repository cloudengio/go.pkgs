// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const (
	// SocketEnvVar is the environment variable that child processes can check
	// for the daemon socket path or address.
	SocketEnvVar = "KEYCHAIN_TEST_SOCKET"
)

// Server is a local daemon that serves an in-memory Plugin over a Unix domain
// socket (or local TCP on systems without Unix domain sockets).
type Server struct {
	plugin   *Plugin
	listener net.Listener
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	path     string
	network  string
}

// StartServer starts a local Server backed by the specified in-memory Plugin.
// If socketPath is empty, a temporary socket path is created.
func StartServer(ctx context.Context, plugin *Plugin, socketPath ...string) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	network := "unix"
	path := ""
	if len(socketPath) > 0 && socketPath[0] != "" {
		path = socketPath[0]
	} else {
		tmpDir, err := os.MkdirTemp("", "keychain-test-server-*")
		if err != nil {
			cancel()
			return nil, fmt.Errorf("creating temp dir for socket: %w", err)
		}
		path = filepath.Join(tmpDir, "sock")
	}

	ln, err := net.Listen(network, path)
	if err != nil {
		// Fallback to local TCP if unix socket fails (e.g., on Windows)
		network = "tcp"
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			cancel()
			return nil, fmt.Errorf("starting listener: %w", err)
		}
		path = ln.Addr().String()
	}

	srv := &Server{
		plugin:   plugin,
		listener: ln,
		ctx:      ctx,
		cancel:   cancel,
		path:     path,
		network:  network,
	}

	srv.wg.Add(1)
	go srv.loop()

	return srv, nil
}

func (s *Server) loop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				return
			}
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer c.Close()
			_ = s.plugin.ServeIO(s.ctx, c, c)
		}(conn)
	}
}

// Address returns the socket path or address of the Server.
func (s *Server) Address() string {
	return s.path
}

// Network returns the network type ("unix" or "tcp").
func (s *Server) Network() string {
	return s.network
}

// Plugin returns the underlying in-memory Plugin.
func (s *Server) Plugin() *Plugin {
	return s.plugin
}

// Close stops the server and cleans up the socket file.
func (s *Server) Close() error {
	s.cancel()
	err := s.listener.Close()
	s.wg.Wait()
	if s.network == "unix" {
		_ = os.Remove(s.path)
		_ = os.Remove(filepath.Dir(s.path))
	}
	return err
}
