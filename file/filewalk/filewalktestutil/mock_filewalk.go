// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package filewalktestutil provides utilities for testing code that uses
// filewalk.FS.
package filewalktestutil

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/file/filewalk"
	"gopkg.in/yaml.v3"
)

// MockFS implements filewalk.FS for testing purposes.
// Note that:
//  1. It does not support soft links.
//  2. It does not support Open on directories, instead, LevelScanner
//     should be used.
//  3. It only supports paths that begin with the root directory passed to
//     NewMockFS.
type MockFS struct {
	root string
	options
	dir dir
}

type dirEntry struct {
	name string
	file *fileEntry
	dir  *dir
}

type fileEntry struct {
	contents []byte
	info     file.Info
}

func (de dirEntry) IsDir() bool {
	return de.dir != nil
}

type dir struct {
	info    file.Info
	entries []dirEntry
}

func findEntry(name string, entries []dirEntry) (dirEntry, bool) {
	for _, e := range entries {
		if e.name == name {
			return e, true
		}
	}
	return dirEntry{}, false
}

func (m *MockFS) lookup(pathname string) (dirEntry, bool) {
	pathname = path.Clean(pathname)
	if pathname == m.root {
		return dirEntry{name: m.root, dir: &m.dir}, true
	}
	r := strings.TrimPrefix(pathname, m.root)
	r = strings.TrimPrefix(r, "/")
	return m.dir.lookup(strings.Split(r, "/"))
}

func (d *dir) lookup(components []string) (dirEntry, bool) {
	switch len(components) {
	case 0:
		// should never get here.
		return dirEntry{}, false
	case 1:
		return findEntry(components[0], d.entries)
	default:
	}
	de, ok := findEntry(components[0], d.entries)
	if !ok {
		return dirEntry{}, false
	}
	return de.dir.lookup(components[1:])
}

type Option func(o *options)

type options struct {
	ymalConfig string
}

func WithYAMLConfig(config string) Option {
	return func(o *options) {
		o.ymalConfig = config
	}
}

func NewMockFS(root string, opts ...Option) (*MockFS, error) {
	m := &MockFS{root: path.Clean(root)}
	for _, opt := range opts {
		opt(&m.options)
	}
	if len(m.ymalConfig) > 0 {
		if err := m.initFromYAML(m.ymalConfig); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (mfs *MockFS) Scheme() string {
	return "mock"
}

func (mfs *MockFS) Open(pathname string) (fs.File, error) {
	pathname = path.Clean(pathname)
	de, ok := mfs.lookup(pathname)
	if !ok || de.IsDir() {
		return nil, os.ErrNotExist
	}
	rd := &filetestutil.BufferCloser{Buffer: bytes.NewBuffer(de.file.contents)}
	return filetestutil.NewFile(rd, &de.file.info), nil
}

func (mfs *MockFS) OpenCtx(_ context.Context, pathname string) (fs.File, error) {
	return mfs.Open(pathname)
}

func (mfs *MockFS) Readlink(ctx context.Context, pathname string) (string, error) {
	return "", fmt.Errorf("soft links are not supported")
}

func (mfs *MockFS) Stat(ctx context.Context, pathname string) (file.Info, error) {
	de, ok := mfs.lookup(pathname)
	if !ok {
		return file.Info{}, os.ErrNotExist
	}
	if de.IsDir() {
		return de.dir.info, nil
	}
	return de.file.info, nil
}

func (mfs *MockFS) Lstat(ctx context.Context, path string) (file.Info, error) {
	return mfs.Stat(ctx, path)
}

func (mfs *MockFS) Join(components ...string) string {
	return path.Join(components...)
}

func (mfs *MockFS) Base(pathname string) string {
	return path.Base(pathname)
}

func (mfs *MockFS) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (mfs *MockFS) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (mfs *MockFS) XAttr(ctx context.Context, pathname string, fi file.Info) (file.XAttr, error) {
	de, ok := mfs.lookup(pathname)
	if !ok || de.IsDir() {
		return fi.Sys().(file.XAttr), os.ErrNotExist
	}
	return file.XAttr{}, nil
}

func (mfs *MockFS) SysXAttr(existing any, merge file.XAttr) any {
	return merge
}

func (mfs *MockFS) LevelScanner(pathname string) filewalk.LevelScanner {
	de, ok := mfs.lookup(pathname)
	if !ok || !de.IsDir() {
		return &scanner{err: os.ErrNotExist}
	}
	return &scanner{entries: de.dir.entries}
}

func (mfs *MockFS) String() string {
	var out strings.Builder
	out.WriteString(mfs.root)
	out.WriteRune('\n')
	printTree(&out, &mfs.dir, 1)
	return out.String()
}

func printTree(out *strings.Builder, d *dir, level int) {
	if d == nil {
		return
	}
	indent := strings.Repeat(" ", level)
	for _, e := range d.entries {
		out.WriteString(indent)
		out.WriteString(e.name)
		out.WriteRune('\n')
		if e.IsDir() {
			printTree(out, e.dir, level+1)
		}
	}
}

type scanner struct {
	entries  []dirEntry
	pos, end int
	err      error
}

func (s *scanner) Contents() []filewalk.Entry {
	return newContents(s.entries[s.pos:s.end])
}

func (s *scanner) Scan(ctx context.Context, n int) bool {
	if s.err != nil {
		return false
	}
	if s.pos >= len(s.entries) {
		return false
	}
	s.pos = s.end
	s.end = s.pos + n
	if s.end > len(s.entries) {
		s.end = len(s.entries)
	}
	return true
}

func (s *scanner) Err() error {
	return s.err
}

func newContents(des []dirEntry) []filewalk.Entry {
	c := make([]filewalk.Entry, len(des))
	for i, de := range des {
		ft := fs.FileMode(0)
		if de.IsDir() {
			ft = fs.ModeDir
		}
		c[i] = filewalk.Entry{Name: de.name, Type: ft}
	}
	return c
}

type fileSpec struct {
	Name     string      `yaml:"name"`
	Size     int64       `yaml:"size"`
	Mode     fs.FileMode `yaml:"mode"`
	Time     time.Time   `yaml:"time"`
	Contents string      `yaml:"contents"`
	UID      int64       `yaml:"uid"`
	GID      int64       `yaml:"gid"`
}

type entrySpec struct {
	File *fileSpec `yaml:"file"`
	Dir  *dirSpec  `yaml:"dir"`
}

type dirSpec struct {
	Name    string      `yaml:"name"`
	UID     int64       `yaml:"uid"`
	GID     int64       `yaml:"gid"`
	Entries []entrySpec `yaml:"entries"`
}

func (mfs *MockFS) initFromYAML(cfg string) error {
	var ds dirSpec
	cfg = strings.ReplaceAll(cfg, "\t", "    ")
	if err := yaml.Unmarshal([]byte(cfg), &ds); err != nil {
		return err
	}
	mfs.dir = *createFromYAML(&ds)
	return nil
}

func createFromYAML(ds *dirSpec) *dir {
	d := &dir{
		info: file.NewInfo(ds.Name, 0, fs.ModeDir, time.Time{},
			file.XAttr{UID: ds.UID, GID: ds.GID}),
	}
	for _, de := range ds.Entries {
		if de.Dir != nil {
			d.entries = append(d.entries, dirEntry{name: de.Dir.Name, dir: createFromYAML(de.Dir)})
			continue
		}
		fe := &fileEntry{
			contents: []byte(de.File.Contents),
			info: file.NewInfo(
				de.File.Name,
				de.File.Size,
				de.File.Mode,
				de.File.Time,
				file.XAttr{UID: de.File.UID, GID: de.File.GID},
			),
		}
		d.entries = append(d.entries, dirEntry{name: de.File.Name, file: fe})
	}
	return d
}
