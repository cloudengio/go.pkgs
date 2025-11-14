// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychain_test

import (
	"context"
	"encoding/base64"
	"os/exec"
	"path/filepath"
	"testing"

	"cloudeng.io/security/keys/keychain"
	"cloudeng.io/security/keys/keychain/plugins"
)

func buildPluginCommand(t *testing.T, binary, sourceCode string) {
	out, err := exec.Command("go", "build", "-o", binary, sourceCode).CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", out)
		t.Fatalf("Failed to build plugin %s: %v", sourceCode, err)
	}
}

func TestExtPlugin(t *testing.T) {
	ctx := context.Background()
	pluginExec := filepath.Join(t.TempDir(), "testplugin")

	buildPluginCommand(t, pluginExec, filepath.Join("testdata", "plugin.go"))

	req := plugins.Request{
		Account: "test_account",
		Keyname: "test_key",
	}
	resp, err := keychain.RunExtPlugin(ctx, pluginExec, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if got, want := resp.Account, req.Account; got != want {
		t.Errorf("expected account %q, got %q", want, got)
	}
	if got, want := resp.Keyname, req.Keyname; got != want {
		t.Errorf("expected keyname %q, got %q", want, got)
	}
	if got, want := resp.Contents, base64.StdEncoding.EncodeToString([]byte(req.Keyname)); got != want {
		t.Errorf("expected contents %q, got %q", want, got)
	}
}
