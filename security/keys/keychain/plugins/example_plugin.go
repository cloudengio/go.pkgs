// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

// This file contains an example implementation of a keychain plugin.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"cloudeng.io/encoding/json/jsonmsgs"
	"cloudeng.io/security/keys/keychain/plugins"
)

var (
	errorFlag    string
	contentsFlag string
	keynameFlag  string
	tempFileFlag string
)

func main() {
	flag.StringVar(&errorFlag, "error", "", "Error message to return in the response")
	flag.StringVar(&contentsFlag, "contents", "", "Contents to return in the response")
	flag.StringVar(&keynameFlag, "keyname", "", "Keyname to respond to")
	flag.StringVar(&tempFileFlag, "tempfile", "", "Temporary file to write/read contents to/from")
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "plugin error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	msgr := jsonmsgs.NewMessager(os.Stdout, io.NopCloser(os.Stdin))

	// 1. Read the request from stdin.
	req, err := plugins.ReadRequest(msgr)
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	// 2. Reject requests whose version is newer than this plugin supports.
	if verr := req.CheckVersion(); verr != nil {
		resp := req.NewResponse(nil, verr)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return plugins.WriteResponse(msgr, *resp)
	}

	var respErr *plugins.Error
	if req.Keyname != keynameFlag {
		respErr = plugins.NewErrorKeyNotFound(req.Keyname)
	}
	if errorFlag != "" {
		respErr = &plugins.Error{
			Message: "error from flag",
			Detail:  errorFlag,
		}
	}

	var contents []byte
	if tempFileFlag != "" {
		if req.Write {
			if err := os.WriteFile(tempFileFlag, req.Contents, 0600); err != nil {
				return fmt.Errorf("failed to write to temp file: %w", err)
			}
		} else {
			contents, err = os.ReadFile(tempFileFlag)
			if err != nil {
				return fmt.Errorf("failed to read from temp file: %w", err)
			}
		}
	} else {
		contents = []byte(contentsFlag)
	}
	resp := req.NewResponse(contents, respErr)
	if err := resp.WithPluginSpecific(req.PluginSpecific); err != nil {
		return fmt.Errorf("failed to create response: %w", err)
	}

	return plugins.WriteResponse(msgr, *resp)
}
