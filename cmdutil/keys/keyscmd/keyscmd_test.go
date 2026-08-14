// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keyscmd_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/keys/keyscmd"
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

	// Non-existent key lookup -> should error
	if _, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "notfound", User: "user"}); err == nil {
		t.Errorf("expected error looking up notfound key")
	}

	// Delete a key
	if err := writer.DeleteKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err != nil {
		t.Fatalf("DeleteKey k1: %v", err)
	}

	// Ensure it is deleted
	_, err = reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"})
	if err == nil {
		t.Fatalf("expected error getting deleted key, got nil")
	}

	// Test ReadKeyInfoFromLocalJSON and ReadKeyInfoFromLocalYAML
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "key.json")
	yamlFile := filepath.Join(tmpDir, "key.yaml")

	if err := reader.SafeWriteKeyInfoJSON(ctx, k2, jsonFile, 0o600); err != nil {
		t.Fatalf("SafeWriteKeyInfoJSON: %v", err)
	}
	if err := reader.SafeWriteKeyInfoYAML(ctx, k2, yamlFile, 0o600); err != nil {
		t.Fatalf("SafeWriteKeyInfoYAML: %v", err)
	}

	readJSONKey, err := writer.ReadKeyInfoFromLocalJSON(ctx, jsonFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalJSON: %v", err)
	}
	if got, want := readJSONKey.ID, "k2"; got != want {
		t.Errorf("readJSONKey.ID = %q, want %q", got, want)
	}

	readYAMLKey, err := writer.ReadKeyInfoFromLocalYAML(ctx, yamlFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalYAML: %v", err)
	}
	if got, want := readYAMLKey.ID, "k2"; got != want {
		t.Errorf("readYAMLKey.ID = %q, want %q", got, want)
	}

	// Read from non-existent file
	if _, err := writer.ReadKeyInfoFromLocalJSON(ctx, filepath.Join(tmpDir, "notfound.json")); err == nil {
		t.Errorf("expected error reading non-existent json file")
	}

	// Read corrupted file
	corruptFile := filepath.Join(tmpDir, "corrupt.json")
	if err := os.WriteFile(corruptFile, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := writer.ReadKeyInfoFromLocalJSON(ctx, corruptFile); err == nil {
		t.Errorf("expected error reading corrupt json file")
	}
}

func TestIsDstSafe(t *testing.T) {
	// Should not fail for normal filenames
	if err := keyscmd.IsDstSafe("testfile.txt"); err != nil {
		t.Errorf("IsDstSafe(testfile.txt) got error: %v", err)
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

	// Extension creation
	ext := keyscmd.NewKeyInfoExtenstion("test-ext", nil)
	if ext.Name() != "test-ext" {
		t.Errorf("ext.Name() = %q, want test-ext", ext.Name())
	}

	// Invalid size
	scInvalidSize := keyscmd.SecretConfig{
		Size:   0,
		Format: keyscmd.SecretFormatHex,
	}
	if _, err := scInvalidSize.New(); err == nil {
		t.Errorf("expected error for size 0")
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

	// When name is stdout ("-")
	stdoutFS := keyscmd.ReadWriteFSWithStdout(mfs, "-")
	if stdoutFS == nil {
		t.Fatalf("ReadWriteFSWithStdout returned nil")
	}

	// Test WriteFSWithStdout and ReadFSWithStdin
	wfs := keyscmd.WriteFSWithStdout(mfs, "-")
	if wfs == nil {
		t.Fatalf("WriteFSWithStdout returned nil")
	}
	rfs := keyscmd.ReadFSWithStdin(mfs, "-")
	if rfs == nil {
		t.Fatalf("ReadFSWithStdin returned nil")
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
}


