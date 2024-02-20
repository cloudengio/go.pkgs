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

func (c *container) Open(filename string) (fs.File, error) {
	if filename != c.filename {
		return nil, fs.ErrNotExist
	}
	return nil, nil
}

//go:embed testdata/hello.txt
var helloBytes []byte

//go:embed testdata/hello.txt
var worldBytes []byte

func TestOpenReadFile(t *testing.T) {
	ctx := context.Background()

	filenameA := path.Join("testdata", "hello.txt")
	filenameB := path.Join("testdata", "world.txt")
	data, err := file.FSReadFile(ctx, filenameA)
	if err != nil {
		t.Error(err)
	}
	if got, want := data, helloBytes; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	dummyA := &container{filename: filenameA, contents: []byte("dummy hello data\n")}
	dummyB := &container{filename: filenameB, contents: []byte("dummy world data\n")}

	ctx = file.ContextWithFS(ctx, dummyA, dummyB)
	data, err = file.FSReadFile(ctx, filenameA)
	if err != nil {
		t.Error(err)
	}
	if got, want := data, dummyA.contents; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	data, err = file.FSReadFile(ctx, filenameB)
	if err != nil {
		t.Error(err)
	}
	if got, want := data, dummyB.contents; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
