// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/security/keys/keychain/plugins"
)

// checkNoDuplicateSegments verifies that no segment (split by ": ") appears
// more than once in the error message. Returns the message for further checks.
func checkNoDuplicateSegments(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
		return ""
	}
	msg := err.Error()
	seen := map[string]int{}
	for part := range strings.SplitSeq(msg, ": ") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		seen[part]++
	}
	for part, count := range seen {
		if count > 1 {
			t.Errorf("segment %q repeated %d times in error message: %q", part, count, msg)
		}
	}
	return msg
}

func TestFS(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "keystore")

	newfs := func(writable bool, args ...string) *plugins.FS {
		t.Helper()
		return plugins.NewFS(pluginPath, writable, nil, args...)
	}

	t.Run("write-and-read", func(t *testing.T) {
		fs := newfs(true, "--tempfile="+tmpFile, "--keyname=my-secret-key")
		key := "my-secret-key"
		secret := []byte("my-super-secret-value")

		// Write the secret.
		if err := fs.WriteFileCtx(ctx, key, secret, 0600); err != nil {
			t.Fatalf("WriteFileCtx failed: %v", err)
		}

		// Read it back.
		readSecret, err := fs.ReadFileCtx(ctx, key)
		if err != nil {
			t.Fatalf("ReadFileCtx failed: %v", err)
		}

		if got, want := string(readSecret), string(secret); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("read-not-found", func(t *testing.T) {
		fs := newfs(false, "--tempfile="+tmpFile, "--keyname=my-secret-key")
		_, err := fs.ReadFileCtx(ctx, "non-existent-key")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected os.ErrNotExist, got %v", err)
		}
	})

	t.Run("write-readonly", func(t *testing.T) {
		fs := newfs(false)
		err := fs.WriteFileCtx(ctx, "any-key", []byte("any-data"), 0600)
		if !errors.Is(err, plugins.ErrReadOnly) {
			t.Errorf("expected ErrReadOnly, got %v", err)
		}
		// Also test non-context WriteFile
		err = fs.WriteFile("any-key", []byte("any-data"), 0600)
		if !errors.Is(err, plugins.ErrReadOnly) {
			t.Errorf("expected ErrReadOnly from WriteFile, got %v", err)
		}
	})

	t.Run("plugin-path-and-logger", func(t *testing.T) {
		fs := newfs(true)
		if got, want := fs.PluginPath(), pluginPath; got != want {
			t.Errorf("PluginPath() = %q, want %q", got, want)
		}
		fsWithLogger := fs.WithLogger(nil)
		if fsWithLogger != fs {
			t.Errorf("WithLogger(nil) should return same pointer")
		}
	})

	t.Run("write-error", func(t *testing.T) {
		fs := newfs(true, "--error=write-failed")
		err := fs.WriteFileCtx(ctx, "any-key", []byte("any-data"), 0600)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "write-failed") {
			t.Errorf("expected error to contain 'write-failed', got %v", err)
		}
	})
}

func TestFSErrorMessages(t *testing.T) {
	ctx := t.Context()

	t.Run("binary-not-found", func(t *testing.T) {
		fs := plugins.NewFS("/nonexistent/plugin-binary", false, nil)
		_, err := fs.ReadFileCtx(ctx, "key")
		msg := checkNoDuplicateSegments(t, err)
		if !strings.Contains(msg, "failed to run plugin") {
			t.Errorf("expected 'failed to run plugin' in error: %q", msg)
		}
	})

	t.Run("write-plugin-error", func(t *testing.T) {
		fs := plugins.NewFS(pluginPath, true, nil, "--error=write-failed")
		err := fs.WriteFileCtx(ctx, "key", []byte("data"), 0600)
		msg := checkNoDuplicateSegments(t, err)
		if !strings.Contains(msg, "write-failed") {
			t.Errorf("expected 'write-failed' in error: %q", msg)
		}
	})

	t.Run("read-plugin-error", func(t *testing.T) {
		fs := plugins.NewFS(pluginPath, false, nil, "--error=read-failed")
		_, err := fs.ReadFileCtx(ctx, "key")
		msg := checkNoDuplicateSegments(t, err)
		if !strings.Contains(msg, "read-failed") {
			t.Errorf("expected 'read-failed' in error: %q", msg)
		}
	})

	t.Run("binary-not-found-write", func(t *testing.T) {
		fs := plugins.NewFS("/nonexistent/plugin-binary", true, nil)
		err := fs.WriteFileCtx(ctx, "key", []byte("data"), 0600)
		msg := checkNoDuplicateSegments(t, err)
		if !strings.Contains(msg, "failed to run plugin") {
			t.Errorf("expected 'failed to run plugin' in error: %q", msg)
		}
	})
}
