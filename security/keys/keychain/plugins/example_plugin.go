// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

// This file contains an example implementation of a keychain plugin.
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

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
	// 1. Read the request from stdin.
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	// 2. Unmarshal the request.
	var req plugins.Request
	if err := json.Unmarshal(input, &req); err != nil {
		// If we can't unmarshal, we can't get the request ID to formulate
		// a valid response.
		return fmt.Errorf("failed to unmarshal request: %w", err)
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
			dec, _ := base64.StdEncoding.DecodeString(req.Contents)
			if err := os.WriteFile(tempFileFlag, []byte(dec), 0600); err != nil {
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
	resp, err := req.NewResponse(contents, respErr, req.SysSpecific)
	if err != nil {
		// This would typically be a JSON marshaling error for the sysSpecific part.
		return fmt.Errorf("failed to create response: %w", err)
	}

	// 5. Marshal the response to JSON.
	output, err := json.Marshal(resp)
	if err != nil {
		// This is an internal error and should not happen with valid data.
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// 6. Write the response to stdout.
	_, err = os.Stdout.Write(output)
	return err
}
