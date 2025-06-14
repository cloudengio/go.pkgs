// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !linux

package largefile

import (
	"context"
	"os"

	"cloudeng.io/sync/errgroup"
)

func reserveSpace(ctx context.Context, fs *os.File, size int64, blockSize, concurrency int) error {
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
	for range concurrency {
		g.Go(func() error {
			for r := range brCh {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if _, err := fs.WriteAt(buf[:r.Size()], r.From); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	return g.Wait()
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
