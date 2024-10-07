// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package keychainfs

import (
	"bytes"
	"io/fs"
	"os/user"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/macos/keychain"
)

// SecureNoteFS implements an fs.ReadFS that reads secure notes from the macOS keychain.
type SecureNoteFS struct {
	options
}

type Option func(*options)

type options struct {
	account string
}

func DefaultAccount() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

func defaultOptions(o *options) {
	o.account = DefaultAccount()
}

func WithAccount(account string) Option {
	return func(o *options) {
		o.account = account
	}
}

func New(opts ...Option) *SecureNoteFS {
	fs := &SecureNoteFS{}
	defaultOptions(&fs.options)
	for _, fn := range opts {
		fn(&fs.options)
	}
	return fs
}

func (fs *SecureNoteFS) Open(name string) (fs.File, error) {
	data, err := keychain.ReadSecureNote(fs.account, name)
	if err != nil {
		return nil, err
	}
	return &nf{name: name, size: len(data), buf: bytes.NewBuffer(data)}, nil
}

type nf struct {
	name string
	size int
	buf  *bytes.Buffer
}

func (f *nf) Stat() (fs.FileInfo, error) {
	var t time.Time
	return file.NewInfo(f.name, int64(f.size), 0, t, nil), nil
}

func (f *nf) Read(p []byte) (int, error) {
	return f.buf.Read(p)
}

func (f *nf) Close() error {
	f.buf.Reset()
	return nil
}

func (fs *SecureNoteFS) ReadFile(name string) ([]byte, error) {
	return keychain.ReadSecureNote(fs.account, name)
}
