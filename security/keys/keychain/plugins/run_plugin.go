// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// RunExtPlugin runs an external keychain plugin with the provided request
// and returns the response. binary is either a command on the PATH or
// an absolute path to the plugin executable.
func RunExtPlugin(ctx context.Context, binary string, req Request, args ...string) (Response, error) {
	if binary == "" {
		return Response{}, fmt.Errorf("plugin binary not specified")
	}
	in := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	enc := json.NewEncoder(in)
	if err := enc.Encode(req); err != nil {
		return Response{Error: &Error{
			Message: "failed to create request",
			Detail:  err.Error(),
		}}, fmt.Errorf("failed to create request: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = in
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	var resp Response
	if err := cmd.Run(); err != nil {
		rerr := &Error{
			Message: "failed to run plugin",
			Detail:  err.Error(),
			Stderr:  stderr.String(),
		}
		return Response{Error: rerr}, fmt.Errorf("failed to run plugin:  %w", rerr)
	}
	if err := json.NewDecoder(stdout).Decode(&resp); err != nil {
		rerr := &Error{
			Message: "failed to decode plugin response",
			Detail:  err.Error(),
			Stderr:  stderr.String(),
		}
		return Response{Error: rerr},
			fmt.Errorf("failed to decode plugin response: %v: %w", err, rerr)
	}
	if resp.ID != req.ID {
		rerr := &Error{
			Message: "response ID does not match request ID",
			Detail:  fmt.Sprintf("response ID %d does not match request ID %d", resp.ID, req.ID),
			Stderr:  stderr.String(),
		}
		return Response{Error: rerr}, fmt.Errorf("response ID %d does not match request ID %d: %w", resp.ID, req.ID, rerr)
	}
	if resp.Error != nil {
		resp.Error.Stderr = stderr.String()
		return resp, nil
	}
	return resp, nil
}
