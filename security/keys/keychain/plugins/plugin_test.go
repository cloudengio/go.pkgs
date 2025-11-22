// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/os/executil"
	"cloudeng.io/security/keys/keychain/plugins"
)

var pluginPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "keychain-plugin-test")
	if err != nil {
		panic(err)
	}
	pluginPath, err = executil.GoBuild(
		context.Background(), filepath.Join(tmpDir, "example-plugin"), "./example_plugin.go")
	if err != nil {
		os.RemoveAll(tmpDir)
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

type sysSpecific struct {
	Account string `json:"account"`
}

func TestExtPlugin(t *testing.T) {
	ctx := t.Context()

	withAccount := sysSpecific{
		Account: "account1",
	}

	req, err := plugins.NewRequest("test_key", withAccount)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	resp, err := plugins.RunExtPlugin(ctx, pluginPath, req, "--contents=my-secret", "--keyname=test_key")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	secret, err := resp.UnmarshalContents()
	if err != nil {
		t.Fatalf("failed to unmarshal contents: %v", err)
	}

	if got, want := secret, "my-secret"; string(got) != want {
		t.Errorf("expected contents %q, got %q", want, got)
	}

	if err := resp.UnmarshalSysSpecific(&withAccount); err != nil {
		t.Fatalf("failed to unmarshal sysSpecific: %v", err)
	}

	if got, want := withAccount.Account, "account1"; got != want {
		t.Errorf("expected account %q, got %q", want, got)
	}
}

func TestKeyNotFound(t *testing.T) {
	ctx := t.Context()

	req, err := plugins.NewRequest("not-my-key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	resp, err := plugins.RunExtPlugin(ctx, pluginPath, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !errors.Is(resp.Error, plugins.ErrKeyNotFound) {
		t.Errorf("expected error to be ErrNotFound, got %v", resp.Error)
	}
	if resp.Error.Detail != "not-my-key" {
		t.Errorf("expected error detail to be 'not-my-key', got %q", resp.Error.Detail)
	}

}

func TestArbitrayError(t *testing.T) {
	ctx := t.Context()

	req, err := plugins.NewRequest("test_key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp, err := plugins.RunExtPlugin(ctx, pluginPath, req, "--error=some error")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}

	if resp.Error.Message != "error from flag" {
		t.Errorf("expected error message 'error from flag', got %q", resp.Error.Message)
	}
	if resp.Error.Detail != "some error" {
		t.Errorf("expected error detail 'some error', got %q", resp.Error.Detail)
	}
}

func TestWriteRead(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()

	tmpFile := filepath.Join(tmpDir, "keystore")
	req, err := plugins.NewWriteRequest("test-key", []byte("my-secret-that-i-just-created"), nil)
	if err != nil {
		t.Fatalf("NewWriteRequest failed: %v", err)
	}
	resp, err := plugins.RunExtPlugin(ctx, pluginPath, req, "--tempfile="+tmpFile, "--keyname=test-key")
	if err != nil {
		t.Logf("response error: %+v", resp.Error)
		t.Fatalf("Run failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}
	if len(resp.Contents) != 0 {
		t.Errorf("expected empty contents, got %q", resp.Contents)
	}

	req, err = plugins.NewRequest("test-key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp, err = plugins.RunExtPlugin(ctx, pluginPath, req, "--tempfile="+tmpFile, "--keyname=test-key")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}
	secret, err := resp.UnmarshalContents()
	if err != nil {
		t.Fatalf("failed to unmarshal contents: %v", err)
	}
	if got, want := secret, "my-secret-that-i-just-created"; string(got) != want {
		t.Errorf("expected contents %q, got %q", want, got)
	}
}
