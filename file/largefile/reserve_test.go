// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"cloudeng.io/errors"
	"cloudeng.io/file/diskusage"
)

func TestReserveSpace(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()
	filename := filepath.Join(td, "testfile")
	os.Remove(filename) // Ensure the file does not exist before the test

	size := int64(1024*1024*10) + 33 // 10 MB
	if err := ReserveSpace(ctx, filename, size, 4096, 4, nil); err != nil {
		t.Fatalf("%v: %v", filename, err)
	}

	err := ReserveSpace(ctx, "any", int64(diskusage.PB), 4096, 4, nil)
	if err == nil || !errors.Is(err, ErrNotEnoughSpace) {
		t.Fatalf("reserveSpace failed: %v", err)
	}

	progressCh := make(chan int64, 100)
	doneCh := make(chan struct{})
	var written int64
	go func() {
		for v := range progressCh {
			atomic.StoreInt64(&written, v)
		}
		close(doneCh)
	}()

	if err := ReserveSpace(ctx, filename, size, 4096, 4, progressCh); err != nil {
		t.Fatalf("%v: %v", filename, err)
	}

	<-doneCh
	if got, want := atomic.LoadInt64(&written), size; got != want {
		t.Errorf("expected %d bytes written, got %d", want, got)
	}
}
