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

// KeyChainPluginName is the default name of the external keychain plugin.
// The plugin should be installed and available in the system PATH.
const KeyChainPluginName = "keychain-plugin"

// RunExtPlugin runs an external keychain plugin with the provided request
// and returns the response. binary is either a command on the PATH or
// an absolute path to the plugin executable. If binary is empty it defaults to
// KeyChainPluginName. The default external plugin can be installed using
// the WithExternalPlugin function.
func RunExtPlugin(ctx context.Context, binary string, req Request, args ...string) (Response, error) {
	if binary == "" {
		binary = KeyChainPluginName
	}
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	enc := json.NewEncoder(in)
	if err := enc.Encode(req); err != nil {
		return Response{Error: &Error{
			Message: "failed to create request",
			Detail:  err.Error(),
		}}, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = stderr
	var resp Response
	if err := cmd.Run(); err != nil {
		return Response{}, fmt.Errorf("plugin failed: %w: %s", err, stderr.String())
	}
	if err := json.NewDecoder(out).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("failed to decode plugin response: %v", err)
	}
	if resp.ID != req.ID {
		return Response{}, fmt.Errorf("response ID %d does not match request ID %d", resp.ID, req.ID)
	}
	return resp, nil
}

// IsExtPluginAvailable checks if the external keychain plugin is available.
func IsExtPluginAvailable() bool {
	_, err := exec.LookPath(KeyChainPluginName)
	return err == nil
}
