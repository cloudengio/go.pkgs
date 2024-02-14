// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"cloudeng.io/file/checkpoint"
)

type chkpt struct {
	*T
	mu     sync.Mutex
	prefix string
}

// NewCheckpointOperation returns a checkpoint.Operation that uses the
// S3.
func NewCheckpointOperation(fs *T) checkpoint.Operation {
	return &chkpt{T: fs}
}

func (c *chkpt) Init(ctx context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.prefix) > 0 {
		return fmt.Errorf("checkpoint operation already initialized")
	}
	if len(prefix) == 0 {
		return fmt.Errorf("prefix must be non-empty")
	}
	c.prefix = ensureIsPrefix(prefix, c.options.delimiter)
	return c.EnsurePrefix(ctx, prefix, 0700)
}

func (c *chkpt) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.prefix) == 0 {
		return fmt.Errorf("checkpoint nit initialized")
	}
	return c.DeleteAll(ctx, c.prefix)
}

func (c *chkpt) Checkpoint(ctx context.Context, label string, data []byte) (id string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sorted, err := c.readAllSorted(ctx)
	if err != nil {
		return "", err
	}
	var next string
	if len(sorted) == 0 {
		next = formatFilename(0, label)
	} else {
		last := sorted[len(sorted)-1]
		lastNum, err := strconv.Atoi(last[:checkpointNumFormatSize])
		if err != nil {
			return "", err
		}
		next = formatFilename(lastNum+1, label)
	}
	err = c.Put(ctx, c.Join(c.prefix, next), 0644, data)
	return next, err
}

func (c *chkpt) Compact(ctx context.Context, label string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, err := c.readAllSorted(ctx)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	last := existing[len(existing)-1]
	data, err := c.Get(ctx, c.Join(c.prefix, last))
	if err != nil {
		return err
	}
	for _, f := range existing {
		if err := c.Delete(ctx, c.Join(c.prefix, f)); err != nil {
			return err
		}
	}
	zero := formatFilename(0, label)
	return c.Put(ctx, c.Join(c.prefix, zero), 0644, data)
}

func (c *chkpt) readAllSorted(ctx context.Context) ([]string, error) {
	sc := c.LevelScanner(c.prefix)
	entries := make([]string, 0, c.options.scanSize)
	for sc.Scan(ctx, c.options.scanSize) {
		contents := sc.Contents()
		for _, c := range contents {
			if c.IsDir() {
				continue
			}
			entries = append(entries, c.Name)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sortByNumberOnly(entries)
	return entries, nil
}

func (c *chkpt) Latest(ctx context.Context) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sorted, err := c.readAllSorted(ctx)
	if err != nil {
		return nil, err
	}
	if len(sorted) == 0 {
		return nil, nil
	}
	last := sorted[len(sorted)-1]
	return c.Get(ctx, c.Join(c.prefix, last))
}

func (c *chkpt) Complete(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	sorted, err := c.readAllSorted(ctx)
	if err != nil {
		return err
	}
	for _, s := range sorted {
		if err := c.Delete(ctx, c.Join(c.prefix, s)); err != nil {
			return err
		}
	}
	return nil
}

func (c *chkpt) Load(ctx context.Context, id string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.Join(c.prefix, id)
	return c.Get(ctx, p)
}

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
