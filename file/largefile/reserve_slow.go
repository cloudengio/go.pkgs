// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !linux

package largefile

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

func reserveSpace(ctx context.Context, fs *os.File, size int64, blockSize, concurrency int, progressCh chan<- int64) error {
	if size <= 0 {
		return nil
	}
	br := NewByteRanges(size, blockSize)
	var buf = make([]byte, blockSize)

	g := &errgroup.T{}
	g = errgroup.WithConcurrency(g, concurrency+1)
	brCh := make(chan ByteRange, concurrency)

	g.Go(func() error {
		defer close(brCh)
		return generator(ctx, br, brCh)
	})

	var written int64

	for range concurrency {
		g.Go(func() error {
			for r := range brCh {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if _, err := fs.WriteAt(buf[:r.Size()], r.From); err != nil {
						if errors.Is(err, syscall.ENOSPC) {
							return fmt.Errorf("%s: %w", fs.Name(), ErrNotEnoughSpace)
						}
						return fmt.Errorf("%s: %w", fs.Name(), err)
					}
					if progressCh != nil {
						select {
						case progressCh <- atomic.AddInt64(&written, r.Size()):
						default:
						}
					}
				}
			}
			return nil
		})
	}
	err := g.Wait()
	if progressCh != nil {
		select {
		case progressCh <- atomic.LoadInt64(&written):
		default:
		}
		close(progressCh)
	}
	return err
}

func generator(ctx context.Context, brs *ByteRanges, ch chan<- ByteRange) error {
	for br := range brs.AllClear(0) {
		select {
		case ch <- br:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
