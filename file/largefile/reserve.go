// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"os"
)

// ReserveSpace creates a file with the specified filename and allocates the
// specified size bytes to it. It verifies that the file was created with the
// requested storage allocated. On systems that support space reservation,
// such as Linux, space is reserved accordingly, on others data is written to
// the file to ensure that the space is allocated. The intent is to ensure that
// a download operations never fails because of insufficient local space once
// it has been initiated.
func ReserveSpace(ctx context.Context, filename string, size int64, blockSize, concurrency int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := reserveSpace(ctx, file, size, blockSize, concurrency); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	nfile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to reopen cache file %s: %w", filename, err)
	}
	defer nfile.Close()
	allocated, err := allocated(file, size)
	if err != nil {
		return err
	}
	if !allocated {
		return fmt.Errorf("file %s was not allocated with size %d", filename, size)
	}
	return nil
}
