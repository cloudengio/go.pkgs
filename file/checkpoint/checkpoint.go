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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cloudeng.io/os/lockedfile"
)

// Operation is the interface for checkpointing an operation.
type Operation interface {
	Init(ctx context.Context, root string) error

	// Clear all existing checkpoints.
	Clear(context.Context) error

	// Compact removes all but the most recent checkpoint and
	// renames as the first checkpoint in the sequence.
	Compact(ctx context.Context, label string) error

	// Checkpoint records the successful completion of a step in the
	// operation.
	Checkpoint(ctx context.Context, label string, data []byte) (id string, err error)

	// Latest reads the latest recorded checkpoint.
	Latest(ctx context.Context) ([]byte, error)

	// Complete removes all checkpoints since the operation is
	// deemed to be have comleted successfully.
	Complete(ctx context.Context) error

	// Load reads the checkpoint with the specified id, the id
	// must have been returned by an earlier call to Checkpoint.
	Load(ctx context.Context, id string) ([]byte, error)
}

type dirop struct {
	dir string
	mu  *lockedfile.Mutex
}

const lockfileName = "lock"

// NewDirectoryOperation returns an implementation of Operation that
// uses a directory on the local file system to	record checkpoints.
// This implementation locks the directory using os.Lockedfile and
// rescans it on each call to Checkpoint to determine the latest entry.
// Consequently it is not well suited to very large numbers of checkpoints.
func NewDirectoryOperation() Operation {
	return &dirop{}

}
func (d *dirop) Init(_ context.Context, dir string) error {
	if len(d.dir) > 0 {
		return fmt.Errorf("already initialized")
	}
	d.dir = dir
	d.mu = lockedfile.MutexAt(filepath.Join(dir, lockfileName))
	return os.MkdirAll(dir, 0755)
}

func (d *dirop) Clear(ctx context.Context) error {
	if len(d.dir) == 0 {
		return fmt.Errorf("not yet initialized")
	}
	unlock, err := d.mu.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	existing, err := readDirSorted(ctx, d.dir)
	if err != nil {
		return err
	}
	for _, f := range existing {
		if err := os.Remove(filepath.Join(d.dir, f)); err != nil {
			return err
		}
	}
	return nil
}

func (d *dirop) Complete(ctx context.Context) error {
	unlock, err := d.mu.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	existing, err := readDirSorted(ctx, d.dir)
	if err != nil {
		return err
	}
	for _, f := range existing {
		if err := os.Remove(filepath.Join(d.dir, f)); err != nil {
			return err
		}
	}
	return nil
}

func (d *dirop) Checkpoint(ctx context.Context, label string, data []byte) (id string, err error) {
	unlock, err := d.mu.Lock()
	if err != nil {
		return "", err
	}
	defer unlock()
	existing, err := readDirSorted(ctx, d.dir)
	if err != nil {
		return "", err
	}
	var next string
	if len(existing) == 0 {
		next = formatFilename(0, label)
	} else {
		prev := existing[len(existing)-1]
		n, err := strconv.Atoi(prev[:checkpointNumFormatSize])
		if err != nil {
			return "", fmt.Errorf("invalid checkpoint filename %q: %v", prev, err)
		}
		next = formatFilename(n+1, label)
	}
	err = os.WriteFile(filepath.Join(d.dir, next), data, 0644) // #nosec G306
	return next, err
}

func (d *dirop) Load(_ context.Context, id string) ([]byte, error) {
	// No need to lock the directory.
	return os.ReadFile(filepath.Join(d.dir, id))
}

func (d *dirop) Latest(ctx context.Context) ([]byte, error) {
	unlock, err := d.mu.Lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	existing, err := readDirSorted(ctx, d.dir)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return nil, nil
	}
	prev := existing[len(existing)-1]
	return os.ReadFile(filepath.Join(d.dir, prev))
}

func (d *dirop) Compact(ctx context.Context, label string) error {
	unlock, err := d.mu.Lock()
	if err != nil {
		return err
	}
	defer unlock()

	existing, err := readDirSorted(ctx, d.dir)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	prev := existing[len(existing)-1]
	data, err := os.ReadFile(filepath.Join(d.dir, prev))
	if err != nil {
		return err
	}
	for _, f := range existing {
		if err := os.Remove(filepath.Join(d.dir, f)); err != nil {
			return err
		}
	}
	zero := formatFilename(0, label)
	return os.WriteFile(filepath.Join(d.dir, zero), data, 0644) // #nosec G306
}

func readDirSorted(ctx context.Context, path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
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
			if n := info.Name(); !info.IsDir() && strings.HasSuffix(n, checkpointSuffix) && len(n) >= (checkpointNumFormatSize+len(checkpointSuffix)) {
				files = append(files, n)
			}
		}
		if err == io.EOF {
			sortByNumberOnly(files)
			return files, nil
		}
	}
}

// sort files by the number prefix, ignoring the label.
func sortByNumberOnly(files []string) {
	sort.Slice(files, func(i, j int) bool {
		return files[i][:checkpointNumFormatSize] < files[j][:checkpointNumFormatSize]
	})
}

const (
	checkpointNumFormat     = "%08d"
	checkpointNumFormatSize = 8
	checkpointSuffix        = ".chk"
)

func formatFilename(n int, label string) string {
	return fmt.Sprintf(checkpointNumFormat+"%s"+checkpointSuffix, n, label)
}
