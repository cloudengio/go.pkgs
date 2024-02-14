// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"
	"sync/atomic"

	"cloudeng.io/errors"
	"cloudeng.io/file/content"
	"cloudeng.io/sync/errgroup"
)

// AsyncWrite represents a store for objects with asynchronous writes. Reads
// are synchronous. The caller must ensure that Finish is called to ensure
// that all writes have completed.
type Async struct {
	fs      content.FS
	written int64
	read    int64
	ch      chan writeRequest
	g       *errgroup.T
	errs    *errors.M
}

type writeRequest struct {
	prefix, name string
	data         []byte
}

func NewAsync(ctx context.Context, fs content.FS, concurrency int) *Async {
	s := &Async{
		fs:   fs,
		ch:   make(chan writeRequest, concurrency),
		g:    &errgroup.T{},
		errs: &errors.M{},
	}
	for i := 0; i < concurrency; i++ {
		s.g.Go(func() error {
			return s.writer(ctx)
		})
	}
	return s
}

func (s *Async) writer(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-s.ch:
			if !ok {
				return nil
			}
			err := write(ctx, s.fs, req.prefix, req.name, req.data, &s.written)
			s.errs.Append(err)
		}
	}
}

// EraseExisting deletes all contents of the store beneath root.
func (s *Async) EraseExisting(ctx context.Context, root string) error {
	return eraseExisting(ctx, s.fs, root)
}

func (s *Async) FS() content.FS {
	return s.fs
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Async) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error) {
	return read(ctx, s.fs, s.fs.Join(prefix, name), &s.read)
}

// Stats returns the number of objects read and written to the store
// since this instance was created.
func (s *Async) Stats() (read, written int64) {
	return atomic.LoadInt64(&s.read), atomic.LoadInt64(&s.written)
}

// Write queues a write request for the specified prefix and name in the store.
// There is no guarantee that the write will have completed when this method
// returns. The error code returned is an indication that the write request
// was queued and will only ever context.Canceled if non-nil.
func (s *Async) Write(ctx context.Context, prefix, name string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.ch <- writeRequest{prefix, name, data}:
		return nil
	}
}

// Finish waits for all queued writes to complete and returns any errors
// encountered during the writes.
func (s *Async) Finish() error {
	close(s.ch)
	errs := s.g.Wait()
	s.errs.Append(errs)
	return s.errs.Err()
}
