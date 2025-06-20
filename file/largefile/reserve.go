// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/file/diskusage"
	"cloudeng.io/sys"
)

var ErrNotEnoughSpace = errors.New("not enough space available for the requested operation")

// ReserveSpace creates a file with the specified filename and allocates the
// specified size bytes to it. It verifies that the file was created with the
// requested storage allocated. On systems that support space reservation,
// such as Linux, space is reserved accordingly, on others data is written to
// the file to ensure that the space is allocated. The intent is to ensure that
// a download operations never fails because of insufficient local space once
// it has been initiated. Progress can be reported via the progressCh channel,
// which will receive updates on the amount of space reserved. The channel
// will be closed when ReserveSpace returns.
func ReserveSpace(ctx context.Context, filename string, size int64, blockSize, concurrency int, progressCh chan<- int64) error {

	availBytes, err := sys.AvailableBytes(filepath.Dir(filename))
	if err != nil {
		return fmt.Errorf("failed to determine available bytes for %s: %w", filename, err)
	}

	if availBytes < size {
		return fmt.Errorf("%s: needs %v, but filesystem has %v: %w", filename, diskusage.Decimal(size), diskusage.Decimal(availBytes), ErrNotEnoughSpace)
	}

	f1, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f1.Close()

	if err := reserveSpace(ctx, f1, size, blockSize, concurrency, progressCh); err != nil {
		return err
	}
	if err := f1.Sync(); err != nil {
		return err
	}

	f2, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to reopen cache file %s: %w", filename, err)
	}
	defer f2.Close()
	allocated, err := allocated(f2, size)
	if err != nil {
		return err
	}
	if !allocated {
		return fmt.Errorf("file %s was not allocated with size %v", filename, diskusage.Decimal(size))
	}

	fi, err := f2.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filename, err)
	}
	if fi.Size() != size {
		return fmt.Errorf("file %s size %d is not equal to requested size %d", filename, fi.Size(), size)
	}
	return nil
}
