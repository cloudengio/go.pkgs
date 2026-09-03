// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"cloudeng.io/encoding/json/jsonmsgs"
	"cloudeng.io/security/keys/keychain/plugins"
)

// Run executes the test plugin CLI logic reading from r and writing to w.
// It parses args, connects to a daemon if specified via flag or env var,
// or handles the request locally.
func Run(ctx context.Context, r io.Reader, w io.Writer, stderr io.Writer, args ...string) error {
	var (
		socketFlag   string
		errorFlag    string
		contentsFlag string
		keynameFlag  string
	)

	fs := flag.NewFlagSet("keychain-test-plugin", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&socketFlag, "socket", os.Getenv(SocketEnvVar), "Socket path/address of the keychain daemon")
	fs.StringVar(&errorFlag, "error", "", "Error detail to return in the response")
	fs.StringVar(&contentsFlag, "contents", "", "Static contents to return in the response")
	fs.StringVar(&keynameFlag, "keyname", "", "Keyname to match for static responses")

	if err := fs.Parse(args); err != nil {
		return err
	}

	msgr := jsonmsgs.NewMessager(w, io.NopCloser(r))

	// 1. Read request from msgr
	req, err := plugins.ReadRequest(msgr)
	if err != nil {
		return fmt.Errorf("reading request: %w", err)
	}

	if verr := req.CheckVersion(); verr != nil {
		resp := req.NewResponse(nil, verr)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return plugins.WriteResponse(msgr, *resp)
	}

	// 2. If a daemon socket is available, forward request to daemon
	if socketFlag != "" {
		network := "unix"
		if strings.Contains(socketFlag, ":") {
			network = "tcp"
		}
		conn, err := net.Dial(network, socketFlag)
		if err != nil {
			return fmt.Errorf("connecting to test server at %s (%s): %w", socketFlag, network, err)
		}
		defer conn.Close()

		connMsgr := jsonmsgs.NewMessager(conn, conn)
		if err := plugins.WriteRequest(connMsgr, req); err != nil {
			return fmt.Errorf("sending request to server: %w", err)
		}

		resp, err := plugins.ReadResponse(connMsgr)
		if err != nil {
			return fmt.Errorf("reading response from server: %w", err)
		}
		return plugins.WriteResponse(msgr, resp)
	}

	// 3. Otherwise handle locally
	if errorFlag != "" {
		resp := req.NewResponse(nil, &plugins.Error{
			Message: "error from flag",
			Detail:  errorFlag,
		})
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return plugins.WriteResponse(msgr, *resp)
	}

	if contentsFlag != "" || keynameFlag != "" {
		if keynameFlag != "" && req.Keyname != keynameFlag {
			resp := req.NewResponse(nil, plugins.NewErrorKeyNotFound(req.Keyname))
			_ = resp.WithPluginSpecific(req.PluginSpecific)
			return plugins.WriteResponse(msgr, *resp)
		}
		resp := req.NewResponse([]byte(contentsFlag), nil)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return plugins.WriteResponse(msgr, *resp)
	}

	// Standalone in-memory fallback
	plugin := New()
	resp := plugin.HandleRequest(ctx, req)
	return plugins.WriteResponse(msgr, resp)
}

// Main is the main entry point for a keychain test plugin executable.
func Main() {
	if err := Run(context.Background(), os.Stdin, os.Stdout, os.Stderr, os.Args[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, "keychain test plugin error: %v\n", err)
		os.Exit(1)
	}
}
