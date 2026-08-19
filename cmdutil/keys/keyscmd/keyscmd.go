// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package keyscmd provides a set of utilities for reading and writing multiple
// keys stored in a single item in a file system using the format used by
// keys.InMemoryKeyStore.
package keyscmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/file"
	"cloudeng.io/file/localfs"
	"gopkg.in/yaml.v3"
)

// IsStdoutStdin returns true if the provided name is "-" or empty, indicating that
// the operation should read from stdin or write to stdout.
func IsStdoutStdin(name string) bool {
	return name == "-" || name == ""
}

// ReadFSWithStdin returns a file.ReadFileFS that reads from stdin if the name is "-" or empty.
func ReadFSWithStdin(fs file.ReadFileFS, name string) file.ReadFileFS {
	if IsStdoutStdin(name) {
		return localfs.AnonymousReadFile{Reader: os.Stdin}
	}
	return fs
}

// WriteFSWithStdout returns a file.WriteFileFS that writes to stdout if the name is "-" or empty.
func WriteFSWithStdout(fs file.WriteFileFS, name string) file.WriteFileFS {
	if IsStdoutStdin(name) {
		return localfs.AnonymousWriteFile{
			Writer: os.Stdout,
		}
	}
	return fs
}

// SafeStdout checks if the provided name refers to Stdout ("-" or empty) and if so,
// verifies that Stdout is piped. If Stdout is not piped, it returns an error.
func IsDstSafe(dst string) error {
	if IsStdoutStdin(dst) && !keys.IsStdoutPiped() {
		return fmt.Errorf("stdout is not piped, cannot write to %q", dst)
	}
	return nil
}

type readWriteFSStdout struct {
	file.ReadFileFS
	io.Writer
}

func (r readWriteFSStdout) WriteFile(_ string, data []byte, _ fs.FileMode) error {
	_, err := r.Write(data)
	return err
}

func (r readWriteFSStdout) WriteFileCtx(_ context.Context, _ string, data []byte, _ fs.FileMode) error {
	_, err := r.Write(data)
	return err
}

// ReadWriterFSWithStdout returns a file.ReadWriteFileFS that reads from fs and writes
// to stdout if the name is "-" or empty.
func ReadWriteFSWithStdout(fs file.ReadWriteFileFS, name string) file.ReadWriteFileFS {
	if name == "-" || name == "" {
		return readWriteFSStdout{
			ReadFileFS: fs,
			Writer:     os.Stdout,
		}
	}
	return fs
}

// CopyContents copies the contents of the source file in srcFS to the
// destination file in dstFS.
func CopyContents(ctx context.Context, srcFS file.ReadFileFS, src string, dstFS file.WriteFileFS, dst string, perm fs.FileMode) error {
	contents, err := srcFS.ReadFileCtx(ctx, src)
	if err != nil {
		return fmt.Errorf("CopyContents: failed to read %v: %w", src, err)
	}
	if err := dstFS.WriteFileCtx(ctx, dst, contents, perm); err != nil {
		return fmt.Errorf("CopyContents: failed to write %v: %w", dst, err)
	}
	return nil
}

func SafeWriteToLocal(ctx context.Context, srcFS file.ReadFileFS, src string, filename string, perm fs.FileMode) error {
	if err := IsDstSafe(filename); err != nil {
		return err
	}
	if err := CopyContents(ctx,
		srcFS, src,
		WriteFSWithStdout(localfs.New(), filename), filename,
		perm); err != nil {
		return fmt.Errorf("failed to write contents to %q: %w", filename, err)
	}
	return nil
}

func ReadFromLocal(ctx context.Context, filename string, dstFS file.WriteFileFS, dst string, perm fs.FileMode) error {
	return CopyContents(ctx,
		ReadFSWithStdin(localfs.New(), filename), filename,
		dstFS, dst,
		perm)
}

func readIMS(ctx context.Context, fs file.ReadFileFS, item string, allowNotExist bool) (*keys.InMemoryKeyStore, error) {
	ims := keys.NewInMemoryKeyStore()
	if err := ims.ReadYAML(ctx, fs, item); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if allowNotExist {
				// if the file doesn't exist, return an empty keystore
				return ims, nil
			}
		}
		return nil, fmt.Errorf("error reading %s from store: %w", item, err)
	}
	return ims, nil
}

func writeIMS(ctx context.Context, fs file.WriteFileFS, item string, ims *keys.InMemoryKeyStore) error {
	if err := ims.WriteYAML(ctx, fs, item, 0600); err != nil {
		return fmt.Errorf("error writing %s to store: %w", item, err)
	}
	return nil
}

var ErrUpdateNotAllowed = errors.New("update not allowed")

// NewKeyReader creates a new KeyReader that reads keys stored using the
// InMemoryKeyStore format using the provided file.ReadFileFS.
func NewKeyReader(fs file.ReadFileFS) KeyReader {
	return KeyReader{fs: fs}
}

// KeyReader provides methods to read keys from a file system in the InMemoryKeyStore
type KeyReader struct {
	fs file.ReadFileFS
}

// GetKeys reads all keys from the specified item in the file system and returns
// them as a slice of keys.Info.
func (r *KeyReader) GetKeys(ctx context.Context, name string) ([]keys.Info, error) {
	ims, err := readIMS(ctx, r.fs, name, false)
	if err != nil {
		return nil, err
	}
	return ims.Keys(), nil
}

func SafeWriteKeyInfoToLocal(ctx context.Context, ki keys.Info, marshal func(any) ([]byte, error), dst string, perm fs.FileMode) error {
	if IsStdoutStdin(dst) {
		redact := func(data []byte) []byte {
			// Redact the key bytes in the token value
			return keys.RedactKeyBytes(data, 4)
		}
		return keys.SafeWriteKeyInfoToStdout(ki, marshal, redact)
	}
	out, err := marshal(ki)
	if err != nil {
		return err
	}
	return localfs.New().WriteFileCtx(ctx, dst, out, perm)
}

func SafeWriteKeyInfoJSON(ctx context.Context, ki keys.Info, dst string, perm fs.FileMode) error {
	return SafeWriteKeyInfoToLocal(ctx, ki, json.Marshal, dst, perm)
}

func SafeWriteKeyInfoYAML(ctx context.Context, ki keys.Info, dst string, perm fs.FileMode) error {
	return SafeWriteKeyInfoToLocal(ctx, ki, yaml.Marshal, dst, perm)
}

var ErrKeyInfoNotFound = errors.New("key info not found")

// GetKey retrieves a specific key from the specified item in the file system
// based on the provided keys.KeySpec. If the key is not found, it returns an error.
func (r *KeyReader) GetKey(ctx context.Context, name string, spec keys.KeySpec) (keys.Info, error) {
	ims, err := readIMS(ctx, r.fs, name, false)
	if err != nil {
		return keys.Info{}, err
	}
	ki, ok := ims.Get(spec.User, spec.ID)
	if !ok {
		return keys.Info{}, fmt.Errorf("secret with user %q and id %q not found in item %s store: %w", spec.User, spec.ID, name, ErrKeyInfoNotFound)
	}
	return ki, nil
}

// KeyWriter provides methods to write keys to a file system in the InMemoryKeyStore
// format.
type KeyWriter struct {
	KeyReader
	fs file.ReadWriteFileFS
}

// NewKeyWriter creates a new KeyWriter that writes keys to an InMemoryKeyStore
// using the provided file.ReadWriteFileFS in the InMemoryKeyStore format.
func NewKeyWriter(fs file.ReadWriteFileFS) KeyWriter {
	return KeyWriter{
		KeyReader: NewKeyReader(fs),
		fs:        fs,
	}
}

// SetKeys adds or updates keys in the specified item in the file system. If update is false,
// it will return an error if any of the keys already exist in the item. If update is true,
// it will overwrite existing keys with the same user and ID.
func (w *KeyWriter) SetKeys(ctx context.Context, name string, update bool, keys ...keys.Info) error {
	ims, err := readIMS(ctx, w.fs, name, true)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if _, ok := ims.Get(key.User, key.ID); ok && !update {
			return fmt.Errorf("secret with user %q and id %q already exists in item %s store: %w", key.User, key.ID, name, ErrUpdateNotAllowed)
		}
		ims.Add(key)
	}
	return writeIMS(ctx, w.fs, name, ims)
}

// ReadKeyInfoFromLocal reads key information from a local file or stdin, unmarshals it
// using the provided unmarshal function, and returns the resulting keys.Info.
func ReadKeyInfoFromLocal(ctx context.Context, filename string, unmarshal func([]byte, any) error) (keys.Info, error) {
	var contents []byte
	if IsStdoutStdin(filename) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return keys.Info{}, fmt.Errorf("reading from stdin: %w", err)
		}
		contents = data
	} else {
		data, err := localfs.New().ReadFileCtx(ctx, filename)
		if err != nil {
			return keys.Info{}, fmt.Errorf("reading from file %q: %w", filename, err)
		}
		contents = data
	}
	var ki keys.Info
	if err := unmarshal(contents, &ki); err != nil {
		return keys.Info{}, fmt.Errorf("unmarshaling key info: %w", err)
	}
	return ki, nil
}

// ReadKeyInfoFromLocalJSON reads key information from a local JSON file or stdin and
// returns the resulting keys.Info.
func ReadKeyInfoFromLocalJSON(ctx context.Context, filename string) (keys.Info, error) {
	return ReadKeyInfoFromLocal(ctx, filename, json.Unmarshal)
}

// ReadKeyInfoFromLocalYAML reads key information from a local YAML file or stdin and
// returns the resulting keys.Info.
func ReadKeyInfoFromLocalYAML(ctx context.Context, filename string) (keys.Info, error) {
	return ReadKeyInfoFromLocal(ctx, filename, yaml.Unmarshal)
}

// DeleteKey removes a specific key from the specified item in the file system based on the
// provided keys.KeySpec. It works by reading all of the existing keys, removing the specified
// key, and then writing the updated list back to the file system.
func (w *KeyWriter) DeleteKey(ctx context.Context, name string, spec keys.KeySpec) error {
	ims, err := readIMS(ctx, w.fs, name, false)
	if err != nil {
		return err
	}
	ims.Delete(spec.User, spec.ID)
	return writeIMS(ctx, w.fs, name, ims)
}

// NewKeyInfoExtension creates a new subcmd.Extension for key info management commands,
// name specifies the name of the command tree and also the name of the template variable
// used to include it in the parent command tree.
func NewKeyInfoExtension(name string, appendFn func(cmd *subcmd.CommandSetYAML) error) subcmd.Extension {
	spec := fmt.Sprintf(keyInfoSubcmdTree, name)
	return subcmd.NewExtension(name, spec, appendFn)
}

// KeySpecFlags defines command-line flags for specifying a key's ID and user.
type KeySpecFlags struct {
	ID   string `subcmd:"key-id,,key id"`
	User string `subcmd:"key-user,,key user"`
}

// KeySpec returns a keys.KeySpec constructed from the KeySpecFlags.
func (f KeySpecFlags) KeySpec() keys.KeySpec {
	return keys.KeySpec{ID: f.ID, User: f.User}
}

// keyInfoSubcmdTree is the subcmd extension tree for managing key info items in a
// keychain/secrets store.
const keyInfoSubcmdTree = `
- name: %s
  summary: manage key info items in a keychain/secrets store, multiple key info items can be stored in a single item. In all cases if input or output is a filename, then "-" or "" will result in stdin or stdout being used as appropriate.
  commands:
    - name: create
      summary: create a new key info, including secret, and write it to <filename>
      arguments:
        - <filename>
    - name: list
      summary: list all key info items in an item
    - name: get
      summary: get a key info from an item from the keychain and write it to <filename>
      arguments:
        - <filename>
    - name: set
      summary: set a key info, read from filename, in an item in the keychain. If the key info already exists it will be overwritten.
      arguments:
        - <filename>
    - name: delete
      summary: delete a key info from an item in the keychain
`

// ExtensionSpec returns the subcmd extension tree for managing key info
// items in a keychain/secrets store, formatted with the provided name.
func ExtensionSpec(name string) string {
	return fmt.Sprintf(keyInfoSubcmdTree, name)
}
