// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keyscmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/keys/keyscmd"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/file"
	"cloudeng.io/file/filetestutil"
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

func (m *mockReadWriteFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.data[name] = append([]byte(nil), data...)
	return nil
}

func (m *mockReadWriteFS) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error {
	return m.WriteFile(name, data, perm)
}

// captureStdout redirects os.Stdout to a pipe for the duration of fn and
// returns everything written to it. Nothing drains the pipe until fn returns,
// so the amount written must fit in the pipe's buffer.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	out, err := filetestutil.CaptureStdout(func() error {
		fn()
		return nil
	})
	if err != nil {
		t.Fatalf("CaptureStdout: %v", err)
	}
	return out
}

// withStdin replaces os.Stdin for the duration of fn with a pipe primed with
// data and closed, so that readers see the data followed by EOF.
func withStdin(t *testing.T, data []byte, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()
	if _, err := w.Write(data); err != nil {
		t.Fatalf("w.Write: %v", err)
	}
	w.Close()
	fn()
}

// wantKeyInfo checks that ki has the expected id, user and token value.
func wantKeyInfo(t *testing.T, ki keys.Info, id, user, token string) {
	t.Helper()
	if got, want := ki.ID, id; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := ki.User, user; got != want {
		t.Errorf("User = %q, want %q", got, want)
	}
	if got, want := string(ki.Token().Value()), token; got != want {
		t.Errorf("Token = %q, want %q", got, want)
	}
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

	// Non-existent key lookup -> should error with ErrKeyInfoNotFound
	if _, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "notfound", User: "user"}); err == nil {
		t.Errorf("expected error looking up notfound key")
	} else if !errors.Is(err, keyscmd.ErrKeyInfoNotFound) {
		t.Errorf("expected ErrKeyInfoNotFound, got %v", err)
	}
}

func TestKeyWriterUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	writer := keyscmd.NewKeyWriter(mfs)
	reader := keyscmd.NewKeyReader(mfs)

	k1 := keys.NewInfo("k1", "user1", []byte("val1"))
	if err := writer.SetKeys(ctx, "store.yaml", false, k1); err != nil {
		t.Fatalf("SetKeys: %v", err)
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

	// Delete a key
	if err := writer.DeleteKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err != nil {
		t.Fatalf("DeleteKey k1: %v", err)
	}

	// Ensure it is deleted
	if _, err := reader.GetKey(ctx, "store.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err == nil {
		t.Fatalf("expected error getting deleted key, got nil")
	} else if !errors.Is(err, keyscmd.ErrKeyInfoNotFound) {
		t.Errorf("expected ErrKeyInfoNotFound, got %v", err)
	}
}

func TestKeyReaderAndWriterMissingFile(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	// Non-existent file lookup in GetKeys
	reader := keyscmd.NewKeyReader(mfs)
	if _, err := reader.GetKeys(ctx, "nonexistent.yaml"); err == nil {
		t.Errorf("expected error getting keys from nonexistent file")
	}

	// Non-existent file in DeleteKey
	writer := keyscmd.NewKeyWriter(mfs)
	if err := writer.DeleteKey(ctx, "nonexistent.yaml", keys.KeySpec{ID: "k1", User: "user1"}); err == nil {
		t.Errorf("expected error deleting key from nonexistent file")
	}
}

func TestSafeWriteReadKeyInfoLocal(t *testing.T) {
	ctx := context.Background()
	k2 := keys.NewInfo("k2", "user2", []byte("val2"))

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
	wantKeyInfo(t, readJSONKey, "k2", "user2", "val2")

	readYAMLKey, err := keyscmd.ReadKeyInfoFromLocalYAML(ctx, yamlFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalYAML: %v", err)
	}
	wantKeyInfo(t, readYAMLKey, "k2", "user2", "val2")

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

	encodings := []struct {
		name      string
		write     func(context.Context, keys.Info, string, fs.FileMode) error
		unmarshal func([]byte, any) error
	}{
		{"JSON", keyscmd.SafeWriteKeyInfoJSON, json.Unmarshal},
		{"YAML", keyscmd.SafeWriteKeyInfoYAML, yaml.Unmarshal},
	}

	// Both "-" and "" name stdout.
	for _, name := range []string{"-", ""} {
		for _, enc := range encodings {
			t.Run(enc.name+"_"+name, func(t *testing.T) {
				out := captureStdout(t, func() {
					if err := enc.write(ctx, ki, name, 0o600); err != nil {
						t.Fatalf("SafeWriteKeyInfo%s to stdout (%q): %v", enc.name, name, err)
					}
				})
				var got keys.Info
				if err := enc.unmarshal(out, &got); err != nil {
					t.Fatalf("%s unmarshal: %v", enc.name, err)
				}
				wantKeyInfo(t, got, "pipeKey", "pipeUser", "secret_value_12345")
			})
		}
	}
}

func TestReadKeyInfoStdin(t *testing.T) {
	ctx := context.Background()
	ki := keys.NewInfo("stdinKey", "stdinUser", []byte("secret_from_stdin"))

	encodings := []struct {
		name    string
		marshal func(any) ([]byte, error)
		read    func(context.Context, string) (keys.Info, error)
	}{
		{"JSON", json.Marshal, keyscmd.ReadKeyInfoFromLocalJSON},
		{"YAML", yaml.Marshal, keyscmd.ReadKeyInfoFromLocalYAML},
	}

	// Both "-" and "" name stdin.
	for _, name := range []string{"-", ""} {
		for _, enc := range encodings {
			t.Run(enc.name+"_"+name, func(t *testing.T) {
				data, err := enc.marshal(ki)
				if err != nil {
					t.Fatalf("%s marshal: %v", enc.name, err)
				}
				withStdin(t, data, func() {
					readKi, err := enc.read(ctx, name)
					if err != nil {
						t.Fatalf("ReadKeyInfoFromLocal%s(%q): %v", enc.name, name, err)
					}
					wantKeyInfo(t, readKi, "stdinKey", "stdinUser", "secret_from_stdin")
				})
			})
		}
	}

	t.Run("corrupted_stdin", func(t *testing.T) {
		withStdin(t, []byte("not valid json or yaml"), func() {
			if _, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, "-"); err == nil {
				t.Errorf("expected error unmarshaling invalid JSON from stdin")
			}
		})
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
	for _, tc := range []struct {
		name    string
		format  keyscmd.SecretFormat
		id      string
		wantLen int // encoded length of a 16 byte secret
	}{
		{"raw", keyscmd.SecretFormatRaw, "rawKey", 16},
		{"hex", keyscmd.SecretFormatHex, "hexKey", 32},       // 16 bytes * 2 hex chars
		{"base64", keyscmd.SecretFormatBase64, "b64Key", 24}, // 16 bytes -> 24 chars
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc := keyscmd.SecretConfig{
				Size:   16,
				Format: tc.format,
				ID:     tc.id,
				User:   "user1",
			}
			ki, err := sc.New()
			if err != nil {
				t.Fatalf("New(): %v", err)
			}
			if got, want := len(ki.Token().Value()), tc.wantLen; got != want {
				t.Errorf("key len = %d, want %d", got, want)
			}
			if got, want := ki.ID, tc.id; got != want {
				t.Errorf("key ID = %q, want %q", got, want)
			}
		})
	}
}

func TestSecretConfigInvalid(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  keyscmd.SecretConfig
	}{
		{"zero size", keyscmd.SecretConfig{Size: 0, Format: keyscmd.SecretFormatHex}},
		{"negative size", keyscmd.SecretConfig{Size: -5, Format: keyscmd.SecretFormatHex}},
		{"invalid format", keyscmd.SecretConfig{Size: 16, Format: keyscmd.SecretFormat(999)}},
	} {
		if _, err := tc.cfg.New(); err == nil {
			t.Errorf("%v: expected an error, got nil", tc.name)
		}
	}
}

func TestSecretConfigFlags(t *testing.T) {
	// SecretConfigFlags conversion
	flags := keyscmd.SecretConfigFlags{
		Size: 24,
		ID:   "from-flags",
		User: "flags-user",
	}
	if err := flags.Format.Set("hex"); err != nil {
		t.Fatalf("flags.Format.Set: %v", err)
	}
	got := flags.SecretConfig()
	want := keyscmd.SecretConfig{
		Size:   24,
		Format: keyscmd.SecretFormatHex,
		ID:     "from-flags",
		User:   "flags-user",
	}
	if got != want {
		t.Errorf("SecretConfig() = %+v, want %+v", got, want)
	}

	// EnumValues check
	if got, want := len(keyscmd.SecretFormatRaw.EnumValues()), 3; got != want {
		t.Errorf("EnumValues len = %d, want %d", got, want)
	}
}

func TestKeyInfoExtension(t *testing.T) {
	appended := false
	ext := keyscmd.NewKeyInfoExtension("test-ext", func(_ *subcmd.CommandSetYAML) error {
		appended = true
		return nil
	})
	if got, want := ext.Name(), "test-ext"; got != want {
		t.Errorf("ext.Name() = %q, want %q", got, want)
	}
	if got, want := ext.YAML(), keyscmd.KeysCmdExtensionSpec("test-ext"); got != want {
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

}

// Both ReadWriteFSWithStdout and WriteFSWithStdout send writes to stdout when
// the name is "-", regardless of the name passed to WriteFile.
func TestWriteFSStdout(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	for _, tc := range []struct {
		name  string
		newFS func() file.WriteFileFS
		data  [2]string
	}{
		{"ReadWriteFSWithStdout",
			func() file.WriteFileFS { return keyscmd.ReadWriteFSWithStdout(mfs, "-") },
			[2]string{"data1", "data2"}},
		{"WriteFSWithStdout",
			func() file.WriteFileFS { return keyscmd.WriteFSWithStdout(mfs, "-") },
			[2]string{"wfs1", "wfs2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The FS must be created while stdout is redirected since it
			// captures os.Stdout at construction time.
			out := captureStdout(t, func() {
				wfs := tc.newFS()
				if wfs == nil {
					t.Fatalf("%v returned nil", tc.name)
				}
				if err := wfs.WriteFile("anything", []byte(tc.data[0]), 0o600); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
				if err := wfs.WriteFileCtx(ctx, "anything", []byte(tc.data[1]), 0o600); err != nil {
					t.Fatalf("WriteFileCtx: %v", err)
				}
			})
			if got, want := string(out), tc.data[0]+tc.data[1]; got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
	}
}

// ReadFSWithStdin reads from stdin when the name is "-".
func TestReadFSStdin(t *testing.T) {
	ctx := context.Background()
	mfs := &mockReadWriteFS{data: make(map[string][]byte)}

	// As above, ReadFSWithStdin captures os.Stdin at construction time.
	withStdin(t, []byte("rfs_input"), func() {
		rfs := keyscmd.ReadFSWithStdin(mfs, "-")
		if rfs == nil {
			t.Fatalf("ReadFSWithStdin returned nil")
		}
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
