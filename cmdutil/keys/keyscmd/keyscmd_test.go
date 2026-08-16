// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keyscmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/keys/keyscmd"
	"cloudeng.io/cmdutil/subcmd"
	"gopkg.in/yaml.v3"
)

type mockReadWriteFS struct {
	data map[string][]byte
}

func (m *mockReadWriteFS) ReadFile(name string) ([]byte, error) {
	if d, ok := m.data[name]; ok {
		return d, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockReadWriteFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	return m.ReadFile(name)
}

func (m *mockReadWriteFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.data[name] = append([]byte(nil), data...)
	return nil
}

func (m *mockReadWriteFS) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error {
	return m.WriteFile(name, data, perm)
}

func TestIsStdoutStdin(t *testing.T) {
	if !keyscmd.IsStdoutStdin("-") {
		t.Errorf("IsStdoutStdin(-) expected true")
	}
	if !keyscmd.IsStdoutStdin("") {
		t.Errorf("IsStdoutStdin(\"\") expected true")
	}
	if keyscmd.IsStdoutStdin("file.txt") {
		t.Errorf("IsStdoutStdin(file.txt) expected false")
	}
}

func TestKeyReaderAndWriter(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	writer := keyscmd.NewKeyWriter(mfs)

	// Set some keys (first-time creation, allowNotExist is true behind the scenes)
	k1 := keys.NewInfo("k1", "user1", []byte("val1"))
	k2 := keys.NewInfo("k2", "user2", []byte("val2"))

	if err := writer.SetKeys(ctx, "store.yaml", false, k1, k2); err != nil {
		t.Fatalf("SetKeys: %v", err)
	}

	// Read them back
	reader := keyscmd.NewKeyReader(mfs)
	keyList, err := reader.GetKeys(ctx, "store.yaml")
	if err != nil {
		t.Fatalf("GetKeys: %v", err)
	}

	if len(keyList) != 2 {
		t.Fatalf("got %d keys, want 2", len(keyList))
	}

	// Retrieve a single key
	gotK1, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"})
	if err != nil {
		t.Fatalf("GetKey k1: %v", err)
	}
	if got, want := gotK1.ID, "k1"; got != want {
		t.Errorf("got key ID %q, want %q", got, want)
	}

	// Try to add k1 again without update=true -> should error
	if err := writer.SetKeys(ctx, "store.yaml", false, k1); err == nil {
		t.Errorf("expected error adding existing key with update=false, got nil")
	} else if !errors.Is(err, keyscmd.ErrUpdateNotAllowed) {
		t.Errorf("expected ErrUpdateNotAllowed, got %v", err)
	}

	// Add k1 with update=true -> should succeed
	k1Updated := keys.NewInfo("k1", "user1", []byte("val1_updated"))
	if err := writer.SetKeys(ctx, "store.yaml", true, k1Updated); err != nil {
		t.Fatalf("SetKeys with update=true: %v", err)
	}
	gotK1Updated, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"})
	if err != nil {
		t.Fatalf("GetKey k1 updated: %v", err)
	}
	if got, want := string(gotK1Updated.Token().Value()), "val1_updated"; got != want {
		t.Errorf("got token %q, want %q", got, want)
	}

	// Non-existent key lookup -> should error with ErrKeyInfoNotFound
	if _, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "notfound", User: "user"}); err == nil {
		t.Errorf("expected error looking up notfound key")
	} else if !errors.Is(err, keyscmd.ErrKeyInfoNotFound) {
		t.Errorf("expected ErrKeyInfoNotFound, got %v", err)
	}

	// Delete a key
	if err := writer.DeleteKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err != nil {
		t.Fatalf("DeleteKey k1: %v", err)
	}

	// Ensure it is deleted
	_, err = reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"})
	if err == nil {
		t.Fatalf("expected error getting deleted key, got nil")
	} else if !errors.Is(err, keyscmd.ErrKeyInfoNotFound) {
		t.Errorf("expected ErrKeyInfoNotFound, got %v", err)
	}

	// Non-existent file lookup in GetKeys
	if _, err := reader.GetKeys(ctx, "nonexistent.yaml"); err == nil {
		t.Errorf("expected error getting keys from nonexistent file")
	}

	// Non-existent file in DeleteKey
	if err := writer.DeleteKey(ctx, "nonexistent.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err == nil {
		t.Errorf("expected error deleting key from nonexistent file")
	}

	// Test SafeWriteKeyInfoJSON and SafeWriteKeyInfoYAML to local file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "key.json")
	yamlFile := filepath.Join(tmpDir, "key.yaml")

	if err := keyscmd.SafeWriteKeyInfoJSON(ctx, k2, jsonFile, 0o600); err != nil {
		t.Fatalf("SafeWriteKeyInfoJSON: %v", err)
	}
	if err := keyscmd.SafeWriteKeyInfoYAML(ctx, k2, yamlFile, 0o600); err != nil {
		t.Fatalf("SafeWriteKeyInfoYAML: %v", err)
	}

	// Verify jsonFile contains valid JSON
	jsonContent, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("ReadFile(jsonFile): %v", err)
	}
	var checkJSON keys.Info
	if err := json.Unmarshal(jsonContent, &checkJSON); err != nil {
		t.Fatalf("json.Unmarshal failed on written JSON file: %v", err)
	}

	// Verify yamlFile contains valid YAML
	yamlContent, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("ReadFile(yamlFile): %v", err)
	}
	var checkYAML keys.Info
	if err := yaml.Unmarshal(yamlContent, &checkYAML); err != nil {
		t.Fatalf("yaml.Unmarshal failed on written YAML file: %v", err)
	}

	readJSONKey, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, jsonFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalJSON: %v", err)
	}
	if got, want := readJSONKey.ID, "k2"; got != want {
		t.Errorf("readJSONKey.ID = %q, want %q", got, want)
	}
	if got, want := string(readJSONKey.Token().Value()), "val2"; got != want {
		t.Errorf("readJSONKey.Token = %q, want %q", got, want)
	}

	readYAMLKey, err := keyscmd.ReadKeyInfoFromLocalYAML(ctx, yamlFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalYAML: %v", err)
	}
	if got, want := readYAMLKey.ID, "k2"; got != want {
		t.Errorf("readYAMLKey.ID = %q, want %q", got, want)
	}
	if got, want := string(readYAMLKey.Token().Value()), "val2"; got != want {
		t.Errorf("readYAMLKey.Token = %q, want %q", got, want)
	}

	// Read from non-existent file
	if _, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, filepath.Join(tmpDir, "notfound.json")); err == nil {
		t.Errorf("expected error reading non-existent json file")
	}
	if _, err := keyscmd.ReadKeyInfoFromLocalYAML(ctx, filepath.Join(tmpDir, "notfound.yaml")); err == nil {
		t.Errorf("expected error reading non-existent yaml file")
	}

	// Read corrupted file
	corruptFile := filepath.Join(tmpDir, "corrupt.json")
	if err := os.WriteFile(corruptFile, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, corruptFile); err == nil {
		t.Errorf("expected error reading corrupt json file")
	}
}

func TestSafeWriteKeyInfoStdout(t *testing.T) {
	ctx := context.Background()
	ki := keys.NewInfo("pipeKey", "pipeUser", []byte("secret_value_12345"))

	for _, name := range []string{"-", ""} {
		t.Run("JSON_"+name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			defer r.Close()

			origStdout := os.Stdout
			os.Stdout = w
			defer func() {
				os.Stdout = origStdout
			}()

			if err := keyscmd.SafeWriteKeyInfoJSON(ctx, ki, name, 0o600); err != nil {
				t.Fatalf("SafeWriteKeyInfoJSON to stdout (%q): %v", name, err)
			}
			w.Close()

			out, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}

			var got keys.Info
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}
			if got.ID != "pipeKey" || got.User != "pipeUser" || string(got.Token().Value()) != "secret_value_12345" {
				t.Errorf("unmarshaled info = %+v, want ki", got)
			}
		})

		t.Run("YAML_"+name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			defer r.Close()

			origStdout := os.Stdout
			os.Stdout = w
			defer func() {
				os.Stdout = origStdout
			}()

			if err := keyscmd.SafeWriteKeyInfoYAML(ctx, ki, name, 0o600); err != nil {
				t.Fatalf("SafeWriteKeyInfoYAML to stdout (%q): %v", name, err)
			}
			w.Close()

			out, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}

			var got keys.Info
			if err := yaml.Unmarshal(out, &got); err != nil {
				t.Fatalf("yaml.Unmarshal: %v", err)
			}
			if got.ID != "pipeKey" || got.User != "pipeUser" || string(got.Token().Value()) != "secret_value_12345" {
				t.Errorf("unmarshaled info = %+v, want ki", got)
			}
		})
	}
}

func TestReadKeyInfoStdin(t *testing.T) {
	ctx := context.Background()
	ki := keys.NewInfo("stdinKey", "stdinUser", []byte("secret_from_stdin"))

	for _, name := range []string{"-", ""} {
		t.Run("JSON_"+name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			defer r.Close()

			origStdin := os.Stdin
			os.Stdin = r
			defer func() {
				os.Stdin = origStdin
			}()

			data, err := json.Marshal(ki)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				t.Fatalf("w.Write: %v", err)
			}
			w.Close()

			readKi, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, name)
			if err != nil {
				t.Fatalf("ReadKeyInfoFromLocalJSON(%q): %v", name, err)
			}
			if readKi.ID != "stdinKey" || readKi.User != "stdinUser" || string(readKi.Token().Value()) != "secret_from_stdin" {
				t.Errorf("ReadKeyInfoFromLocalJSON = %+v, want ki", readKi)
			}
		})

		t.Run("YAML_"+name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			defer r.Close()

			origStdin := os.Stdin
			os.Stdin = r
			defer func() {
				os.Stdin = origStdin
			}()

			data, err := yaml.Marshal(ki)
			if err != nil {
				t.Fatalf("yaml.Marshal: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				t.Fatalf("w.Write: %v", err)
			}
			w.Close()

			readKi, err := keyscmd.ReadKeyInfoFromLocalYAML(ctx, name)
			if err != nil {
				t.Fatalf("ReadKeyInfoFromLocalYAML(%q): %v", name, err)
			}
			if readKi.ID != "stdinKey" || readKi.User != "stdinUser" || string(readKi.Token().Value()) != "secret_from_stdin" {
				t.Errorf("ReadKeyInfoFromLocalYAML = %+v, want ki", readKi)
			}
		})
	}

	t.Run("corrupted_stdin", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		defer r.Close()

		origStdin := os.Stdin
		os.Stdin = r
		defer func() {
			os.Stdin = origStdin
		}()

		if _, err := w.Write([]byte("not valid json or yaml")); err != nil {
			t.Fatalf("w.Write: %v", err)
		}
		w.Close()

		if _, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, "-"); err == nil {
			t.Errorf("expected error unmarshaling invalid JSON from stdin")
		}
	})
}

func TestIsDstSafe(t *testing.T) {
	// Should not fail for normal filenames
	if err := keyscmd.IsDstSafe("testfile.txt"); err != nil {
		t.Errorf("IsDstSafe(testfile.txt) got error: %v", err)
	}

	// When stdout is piped (via os.Pipe), IsDstSafe("-") and IsDstSafe("") should succeed
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()

	origStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	if err := keyscmd.IsDstSafe("-"); err != nil {
		t.Errorf("IsDstSafe(-) on pipe got error: %v", err)
	}
	if err := keyscmd.IsDstSafe(""); err != nil {
		t.Errorf("IsDstSafe(\"\") on pipe got error: %v", err)
	}
}

func TestCopyContents(t *testing.T) {
	ctx := context.Background()
	srcFS := &mockReadWriteFS{
		data: map[string][]byte{
			"src.txt": []byte("hello copy contents"),
		},
	}
	dstFS := &mockReadWriteFS{
		data: make(map[string][]byte),
	}

	if err := keyscmd.CopyContents(ctx, srcFS, "src.txt", dstFS, "dst.txt", 0o600); err != nil {
		t.Fatalf("CopyContents: %v", err)
	}

	if got, want := string(dstFS.data["dst.txt"]), "hello copy contents"; got != want {
		t.Errorf("dst.txt content = %q, want %q", got, want)
	}

	// Source not found
	if err := keyscmd.CopyContents(ctx, srcFS, "nonexistent.txt", dstFS, "dst2.txt", 0o600); err == nil {
		t.Errorf("expected error copying nonexistent source file")
	}
}

func TestSafeWriteToLocalAndReadFromLocal(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "local_test.txt")

	srcFS := &mockReadWriteFS{
		data: map[string][]byte{
			"orig.txt": []byte("local roundtrip data"),
		},
	}

	// Test SafeWriteToLocal to an actual file
	if err := keyscmd.SafeWriteToLocal(ctx, srcFS, "orig.txt", localFile, 0o600); err != nil {
		t.Fatalf("SafeWriteToLocal: %v", err)
	}

	// Verify file on local disk
	contents, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", localFile, err)
	}
	if !bytes.Equal(contents, []byte("local roundtrip data")) {
		t.Errorf("local file contents = %q, want %q", string(contents), "local roundtrip data")
	}

	// SafeWriteToLocal with missing source file
	if err := keyscmd.SafeWriteToLocal(ctx, srcFS, "missing.txt", localFile, 0o600); err == nil {
		t.Errorf("expected error writing missing source file")
	}

	// Test ReadFromLocal reading from that local file into a target FS
	dstFS := &mockReadWriteFS{
		data: make(map[string][]byte),
	}
	if err := keyscmd.ReadFromLocal(ctx, localFile, dstFS, "target.txt", 0o600); err != nil {
		t.Fatalf("ReadFromLocal: %v", err)
	}

	if got, want := string(dstFS.data["target.txt"]), "local roundtrip data"; got != want {
		t.Errorf("target.txt contents = %q, want %q", got, want)
	}

	// ReadFromLocal with missing local file
	if err := keyscmd.ReadFromLocal(ctx, filepath.Join(tmpDir, "missing_local.txt"), dstFS, "target2.txt", 0o600); err == nil {
		t.Errorf("expected error reading nonexistent local file")
	}

	// Test SafeWriteToLocal to pipe ("-") and ReadFromLocal from pipe ("-")
	t.Run("pipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		defer r.Close()

		origStdout := os.Stdout
		origStdin := os.Stdin
		os.Stdout = w
		os.Stdin = r
		defer func() {
			os.Stdout = origStdout
			os.Stdin = origStdin
		}()

		// Write key store / key data to stdout pipe using SafeWriteToLocal with "-"
		pipeSrcFS := &mockReadWriteFS{
			data: map[string][]byte{
				"key_in_pipe.yaml": []byte("- key_id: pipe_key\n  user: pipe_user\n  token: pipe_secret\n"),
			},
		}

		if err := keyscmd.SafeWriteToLocal(ctx, pipeSrcFS, "key_in_pipe.yaml", "-", 0o600); err != nil {
			t.Fatalf("SafeWriteToLocal to stdout pipe: %v", err)
		}
		// Close writing end of pipe so EOF can be reached when reading
		w.Close()

		pipeDstFS := &mockReadWriteFS{
			data: make(map[string][]byte),
		}
		if err := keyscmd.ReadFromLocal(ctx, "-", pipeDstFS, "read_from_pipe.yaml", 0o600); err != nil {
			t.Fatalf("ReadFromLocal from stdin pipe: %v", err)
		}

		// Verify the content read from pipe can be unmarshaled by ReadYAML into InMemoryKeyStore
		ims := keys.NewInMemoryKeyStore()
		if err := ims.ReadYAML(ctx, pipeDstFS, "read_from_pipe.yaml"); err != nil {
			t.Fatalf("ims.ReadYAML from pipe output: %v", err)
		}
		gotKey, ok := ims.Get("pipe_user", "pipe_key")
		if !ok {
			t.Fatalf("expected pipe_key to be found in ims")
		}
		if got, want := string(gotKey.Token().Value()), "pipe_secret"; got != want {
			t.Errorf("token = %q, want %q", got, want)
		}
	})
}

func TestSecretConfig(t *testing.T) {
	// Raw
	scRaw := keyscmd.SecretConfig{
		Size:   16,
		Format: keyscmd.SecretFormatRaw,
		ID:     "rawKey",
		User:   "user1",
	}
	kiRaw, err := scRaw.New()
	if err != nil {
		t.Fatalf("scRaw.New(): %v", err)
	}
	if len(kiRaw.Token().Value()) != 16 {
		t.Errorf("raw key len = %d, want 16", len(kiRaw.Token().Value()))
	}
	if got, want := kiRaw.ID, "rawKey"; got != want {
		t.Errorf("raw key ID = %q, want %q", got, want)
	}

	// Hex
	scHex := keyscmd.SecretConfig{
		Size:   16,
		Format: keyscmd.SecretFormatHex,
		ID:     "hexKey",
		User:   "user1",
	}
	kiHex, err := scHex.New()
	if err != nil {
		t.Fatalf("scHex.New(): %v", err)
	}
	if len(kiHex.Token().Value()) != 32 { // 16 bytes * 2 hex chars
		t.Errorf("hex key len = %d, want 32", len(kiHex.Token().Value()))
	}

	// Base64
	scB64 := keyscmd.SecretConfig{
		Size:   16,
		Format: keyscmd.SecretFormatBase64,
		ID:     "b64Key",
		User:   "user1",
	}
	kiB64, err := scB64.New()
	if err != nil {
		t.Fatalf("scB64.New(): %v", err)
	}
	if len(kiB64.Token().Value()) != 24 { // 16 bytes encoded base64 -> 24 chars
		t.Errorf("b64 key len = %d, want 24", len(kiB64.Token().Value()))
	}

	// SecretConfigFlags conversion
	flags := keyscmd.SecretConfigFlags{
		Size: 24,
		ID:   "from-flags",
		User: "flags-user",
	}
	if err := flags.Format.Set("hex"); err != nil {
		t.Fatalf("flags.Format.Set: %v", err)
	}
	scFromFlags := flags.SecretConfig()
	if scFromFlags.Size != 24 || scFromFlags.ID != "from-flags" || scFromFlags.User != "flags-user" || scFromFlags.Format != keyscmd.SecretFormatHex {
		t.Errorf("SecretConfig() = %+v, unexpected fields", scFromFlags)
	}

	// Invalid size
	scInvalidSize := keyscmd.SecretConfig{
		Size:   0,
		Format: keyscmd.SecretFormatHex,
	}
	if _, err := scInvalidSize.New(); err == nil {
		t.Errorf("expected error for size 0")
	}

	scNegativeSize := keyscmd.SecretConfig{
		Size:   -5,
		Format: keyscmd.SecretFormatHex,
	}
	if _, err := scNegativeSize.New(); err == nil {
		t.Errorf("expected error for negative size")
	}

	// Invalid format
	scInvalidFmt := keyscmd.SecretConfig{
		Size:   16,
		Format: keyscmd.SecretFormat(999),
	}
	if _, err := scInvalidFmt.New(); err == nil {
		t.Errorf("expected error for invalid format")
	}

	// EnumValues check
	ev := keyscmd.SecretFormatRaw.EnumValues()
	if len(ev) != 3 {
		t.Errorf("EnumValues len = %d, want 3", len(ev))
	}
}

func TestKeyInfoExtension(t *testing.T) {
	appended := false
	ext := keyscmd.NewKeyInfoExtension("test-ext", func(cmd *subcmd.CommandSetYAML) error {
		appended = true
		return nil
	})
	if got, want := ext.Name(), "test-ext"; got != want {
		t.Errorf("ext.Name() = %q, want %q", got, want)
	}
	if got, want := ext.YAML(), keyscmd.KeyInfoSubcmdTree; got != want {
		t.Errorf("ext.YAML() = %q, want %q", got, want)
	}
	if err := ext.Set(nil); err != nil {
		t.Fatalf("ext.Set: %v", err)
	}
	if !appended {
		t.Errorf("expected appendFn to have been called")
	}
}

func TestKeySpecFlags(t *testing.T) {
	flags := keyscmd.KeySpecFlags{
		ID:   "my-id",
		User: "my-user",
	}
	spec := flags.KeySpec()
	if got, want := spec.ID, "my-id"; got != want {
		t.Errorf("spec.ID = %q, want %q", got, want)
	}
	if got, want := spec.User, "my-user"; got != want {
		t.Errorf("spec.User = %q, want %q", got, want)
	}
}

func TestReadWriteFSStdout(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	// When name is not stdout/stdin
	normalFS := keyscmd.ReadWriteFSWithStdout(mfs, "somefile.yaml")
	if err := normalFS.WriteFile("somefile.yaml", []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := normalFS.WriteFileCtx(ctx, "somefile2.yaml", []byte("content2"), 0o600); err != nil {
		t.Fatalf("WriteFileCtx: %v", err)
	}

	// Normal names
	wfsNorm := keyscmd.WriteFSWithStdout(mfs, "norm.txt")
	if wfsNorm == nil {
		t.Fatalf("WriteFSWithStdout norm returned nil")
	}
	rfsNorm := keyscmd.ReadFSWithStdin(mfs, "norm.txt")
	if rfsNorm == nil {
		t.Fatalf("ReadFSWithStdin norm returned nil")
	}

	// Test writing to stdoutFS ("-")
	t.Run("stdoutFS_write", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		defer r.Close()

		origStdout := os.Stdout
		os.Stdout = w
		defer func() {
			os.Stdout = origStdout
		}()

		stdoutFS := keyscmd.ReadWriteFSWithStdout(mfs, "-")
		if stdoutFS == nil {
			t.Fatalf("ReadWriteFSWithStdout returned nil")
		}

		if err := stdoutFS.WriteFile("anything", []byte("data1"), 0o600); err != nil {
			t.Fatalf("stdoutFS.WriteFile: %v", err)
		}
		if err := stdoutFS.WriteFileCtx(ctx, "anything", []byte("data2"), 0o600); err != nil {
			t.Fatalf("stdoutFS.WriteFileCtx: %v", err)
		}
		w.Close()

		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if got, want := string(out), "data1data2"; got != want {
			t.Errorf("stdoutFS output = %q, want %q", got, want)
		}
	})

	// Test writing to wfs ("-")
	t.Run("wfs_write", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		defer r.Close()

		origStdout := os.Stdout
		os.Stdout = w
		defer func() {
			os.Stdout = origStdout
		}()

		wfs := keyscmd.WriteFSWithStdout(mfs, "-")
		if wfs == nil {
			t.Fatalf("WriteFSWithStdout returned nil")
		}

		if err := wfs.WriteFile("anything", []byte("wfs1"), 0o600); err != nil {
			t.Fatalf("wfs.WriteFile: %v", err)
		}
		if err := wfs.WriteFileCtx(ctx, "anything", []byte("wfs2"), 0o600); err != nil {
			t.Fatalf("wfs.WriteFileCtx: %v", err)
		}
		w.Close()

		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if got, want := string(out), "wfs1wfs2"; got != want {
			t.Errorf("wfs output = %q, want %q", got, want)
		}
	})

	// Test reading from rfs ("-")
	t.Run("rfs_read", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		defer r.Close()

		origStdin := os.Stdin
		os.Stdin = r
		defer func() {
			os.Stdin = origStdin
		}()

		rfs := keyscmd.ReadFSWithStdin(mfs, "-")
		if rfs == nil {
			t.Fatalf("ReadFSWithStdin returned nil")
		}

		if _, err := w.Write([]byte("rfs_input")); err != nil {
			t.Fatalf("w.Write: %v", err)
		}
		w.Close()

		data, err := rfs.ReadFile("anything")
		if err != nil {
			t.Fatalf("rfs.ReadFile: %v", err)
		}
		if got, want := string(data), "rfs_input"; got != want {
			t.Errorf("rfs output = %q, want %q", got, want)
		}

		dataCtx, err := rfs.ReadFileCtx(ctx, "anything")
		if err != nil {
			t.Fatalf("rfs.ReadFileCtx: %v", err)
		}
		// Second read returns empty because stdin reached EOF
		if len(dataCtx) != 0 {
			t.Errorf("expected empty data at EOF, got %q", string(dataCtx))
		}
	})
}



