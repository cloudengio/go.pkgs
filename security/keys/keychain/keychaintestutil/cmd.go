// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

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
	fs.StringVar(&socketFlag, "socket", os.Getenv(SocketEnvVar), "Socket address of the running keychain test server")
	fs.StringVar(&errorFlag, "error", "", "Error detail to return in the response")
	fs.StringVar(&contentsFlag, "contents", "", "Static contents to return in the response")
	fs.StringVar(&keynameFlag, "keyname", "", "Keyname to match for static responses")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 1. Read request from r
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading request: %w", err)
	}

	var req plugins.Request
	if err := json.Unmarshal(input, &req); err != nil {
		return fmt.Errorf("unmarshaling request: %w", err)
	}

	if verr := req.CheckVersion(); verr != nil {
		resp := req.NewResponse(nil, verr)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return json.NewEncoder(w).Encode(resp)
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

		if _, err := conn.Write(input); err != nil {
			return fmt.Errorf("sending request to server: %w", err)
		}

		respBytes, err := io.ReadAll(conn)
		if err != nil {
			return fmt.Errorf("reading response from server: %w", err)
		}
		_, err = w.Write(respBytes)
		return err
	}

	// 3. Otherwise handle locally
	if errorFlag != "" {
		resp := req.NewResponse(nil, &plugins.Error{
			Message: "error from flag",
			Detail:  errorFlag,
		})
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return json.NewEncoder(w).Encode(resp)
	}

	if contentsFlag != "" || keynameFlag != "" {
		if keynameFlag != "" && req.Keyname != keynameFlag {
			resp := req.NewResponse(nil, plugins.NewErrorKeyNotFound(req.Keyname))
			_ = resp.WithPluginSpecific(req.PluginSpecific)
			return json.NewEncoder(w).Encode(resp)
		}
		resp := req.NewResponse([]byte(contentsFlag), nil)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return json.NewEncoder(w).Encode(resp)
	}

	// Standalone in-memory fallback
	plugin := New()
	resp := plugin.HandleRequest(ctx, req)
	return json.NewEncoder(w).Encode(resp)
}

// Main is the main entry point for a keychain test plugin executable.
func Main() {
	if err := Run(context.Background(), os.Stdin, os.Stdout, os.Stderr, os.Args[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, "keychain test plugin error: %v\n", err)
		os.Exit(1)
	}
}
