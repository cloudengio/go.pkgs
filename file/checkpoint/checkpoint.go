// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package checkpoint provides a mechanism for checkpointing the
// state of an ongoing operation. An operation is defined as any
// application activity that can be meaningfully broken into smaller
// steps and that can be resumed from one of those steps. The record
// of the successful completion of each step is recorded as a 'checkpoint'.
package checkpoint

import (
	"context"
	"io"
	"os"
)

type Operation interface {
	Checkpoint(ctx context.Context, data []byte) (id string, err error)
	Load(ctx context.Context, id string) ([]byte, error)
	Latest(ctx context.Context) ([]byte, error)
}

type dirop struct {
	dir string
}

func NewDirectoryOperation(dir string) (Operation, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

}

func (d *dirop) Checkpoint(ctx context.Context, data []byte) (id string, err error) {
	panic("not implemented")
}

func (d *dirop) Load(id string) error {
	panic("not implemented")
}

func (d *dirop) Latest(ctx context.Context) ([]byte, error) {
	panic("not implemented")
}

func readDirSorted(ctx context.Context, path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanSize := 50
	files := make([]string, 0, 50)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		infos, err := f.ReadDir(scanSize)
		for _, info := range infos {
			if !info.IsDir() {
				files = append(files, info.Name())
			}
		}
		if err == io.EOF {
			return files, nil
		}
	}
}
