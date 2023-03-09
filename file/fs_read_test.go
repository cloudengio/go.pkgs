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
	contents []byte
}

func (c *container) New() fs.ReadFileFS {
	return &container{}
}

func (c *container) ReadFile(name string) ([]byte, error) {
	return c.contents, nil
}

func (c *container) Open(name string) (fs.File, error) {
	return nil, nil
}

//go:embed testdata/hello.txt
var testFSBytes []byte

func TestOpenReadFile(t *testing.T) {
	ctx := context.Background()

	data, err := file.FSReadFile(ctx, path.Join("testdata", "hello.txt"))
	if err != nil {
		t.Error(err)
	}
	if got, want := data, testFSBytes; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	dummy := &container{contents: []byte("dummy data\n")}
	ctx = file.ContextWithFS(ctx, dummy)
	data, err = file.FSReadFile(ctx, path.Join("testdata", "hello.txt"))
	if err != nil {
		t.Error(err)
	}
	if got, want := data, dummy.contents; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
