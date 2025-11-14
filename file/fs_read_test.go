// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file_test

import (
	"bytes"
	"context"
	_ "embed"
	"io/fs"
	"path"
	"testing"

	"cloudeng.io/file"
)

type container struct {
	filename string
	contents []byte
}

func (c *container) ReadFile(filename string) ([]byte, error) {
	if filename != c.filename {
		return nil, fs.ErrNotExist
	}
	return c.contents, nil
}

func (c *container) ReadFileCtx(_ context.Context, filename string) ([]byte, error) {
	if filename != c.filename {
		return nil, fs.ErrNotExist
	}
	return c.contents, nil
}

func (c *container) Open(filename string) (fs.File, error) {
	if filename != c.filename {
		return nil, fs.ErrNotExist
	}
	return nil, nil
}

func (c *container) OpenCtx(_ context.Context, filename string) (fs.File, error) {
	if filename != c.filename {
		return nil, fs.ErrNotExist
	}
	return nil, nil
}

//go:embed testdata/hello.txt
var helloBytes []byte

//go:embed testdata/world.txt
var worldBytes []byte

func TestOpenReadFile(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name          string
		contents      []byte
		dummyContents []byte
	}{
		{path.Join("testdata", "hello.txt"), helloBytes, []byte("dummy hello data\n")},
		{path.Join("testdata", "world.txt"), worldBytes, []byte("dummy world data\n")},
	} {
		data, err := file.FSReadFile(ctx, tc.name)
		if err != nil {
			t.Error(err)
		}
		if got, want := data, tc.contents; !bytes.Equal(got, want) {
			t.Errorf("got %s, want %s", got, want)
		}
		dummy := &container{filename: tc.name, contents: tc.dummyContents}
		ctx = file.ContextWithFS(ctx, dummy)
		data, err = file.FSReadFile(ctx, tc.name)
		if err != nil {
			t.Error(err)
		}
		if got, want := data, tc.contents; bytes.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := data, tc.dummyContents; !bytes.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
