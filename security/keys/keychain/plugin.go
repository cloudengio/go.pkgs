// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package keychain provides functionality to interact with a local keychain.
// It is intended to be used to retrieve a secret that's used to encrypt/decrypt
// all other api tokens. A plugin is used to avoid the need to reauthenticate
// with the OS every time an application that needs the key is recompiled during
// development. For production applications the keychain should be accessed
// directly via the Plugin method.
package keychain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"cloudeng.io/security/keys/keychain/plugins"
)

// RunExtPlugin runs an external keychain plugin with the provided request
// and returns the response. binary is either a command on the PATH or
// an absolute path to the plugin executable. If binary is empty it defaults to
// KeyChainPluginName. The default external plugin can be installed using
// the WithExternalPlugin function.
func RunExtPlugin(ctx context.Context, binary string, req plugins.Request) (plugins.Response, error) {
	if binary == "" {
		binary = KeyChainPluginName
	}
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	enc := json.NewEncoder(in)
	if err := enc.Encode(req); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = in
	cmd.Stdout = out
	var resp plugins.Response
	if err := cmd.Run(); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	if err := json.NewDecoder(out).Decode(&resp); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	return resp, nil
}

// RunPlugin executes the keychain plugin compiled into the running application.
func RunPlugin(ctx context.Context, req plugins.Request) (plugins.Response, error) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	enc := json.NewEncoder(in)
	if err := enc.Encode(req); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	if err := plugins.Plugin(in, out); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	var resp plugins.Response
	if err := json.NewDecoder(out).Decode(&resp); err != nil {
		return plugins.Response{Error: err.Error()}, err
	}
	return resp, nil
}

// isGoRun checks if the current process was started via `go run`. It uses
// a simple heuristic of checking if the executable name in os.Args[0]
// contains "go-run".
func isGoRun() bool {
	return strings.Contains(os.Args[0], "go-run")
}

// RunAvailablePlugin decides whether to use the external plugin or the
// compiled-in plugin based on whether the application is running via `go run`.
func RunAvailablePlugin(ctx context.Context, req plugins.Request) (plugins.Response, error) {
	if isGoRun() {
		return RunExtPlugin(ctx, KeyChainPluginName, req)
	}
	return RunPlugin(ctx, req)
}

// IsExtPluginAvailable checks if the external keychain plugin is available.
func IsExtPluginAvailable(ctx context.Context) bool {
	_, err := exec.LookPath(KeyChainPluginName)
	return err == nil
}

// GetKey retrieves a key from the keychain using the specified plugin (extPluginPath)
// if the application is running via `go run`, or directly if it is a compiled binary.
func GetKey(ctx context.Context, account, keyname string) ([]byte, error) {
	resp, err := RunAvailablePlugin(ctx, plugins.Request{
		Account: account,
		Keyname: keyname,
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	if resp.Contents == "" {
		return nil, errors.New("no contents returned from keychain plugin")
	}
	return base64.StdEncoding.DecodeString(resp.Contents)

}

// SetKey sets a key in the keychain using the specified plugin (extPluginPath).
// If the application is running via `go run`, it uses the external plugin;
// otherwise, it uses the compiled-in plugin.
// The key is written as base64 encoded contents.
// If the key already exists, it will be overwritten.
// The account can be empty, in which case the default account is used.
// The keyname must not be empty.
func SetKey(ctx context.Context, account, keyname string, contents []byte) error {
	encodedContents := base64.StdEncoding.EncodeToString(contents)
	resp, err := RunAvailablePlugin(ctx, plugins.Request{
		Account:  account,
		Keyname:  keyname,
		WriteKey: true,
		Contents: encodedContents,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

const KeyChainPluginName = "keychain_plugin_cmd"

// GetKey retrieves a key from the keychain using the specified account and keyname.
const pkgName = "cloudeng.io/security/keys/keychain/" + KeyChainPluginName + "@latest"

// DevelopmentPluginBuildCommand returns an exec.Cmd that builds the keychain
// plugin for the current OS and installs it as <dir>/<name>. If name
// is empty, it defaults to DefaultExtPluginName. The returned command
// can be executed to build the plugin, and the second return value is the
// location where the plugin will be installed.
func ExtPluginBuildCommand(ctx context.Context) *exec.Cmd {
	return exec.CommandContext(ctx, "go", "install", pkgName)
}

// WithExternalPlugin builds the external plugin if the application is running
// via `go run`. It uses the ExtPluginBuildCommand to build the plugin.
// If the build fails, it returns an error with the output of the build command.
// If the application is not running via `go run`, it does nothing and returns nil.
// This function is intended to be called at the start of the application
// to ensure that the external plugin is built and available for use.
func WithExternalPlugin(ctx context.Context, extPluginPath string) error {
	if !isGoRun() {
		return nil
	}
	out, err := ExtPluginBuildCommand(ctx).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	return nil
}
