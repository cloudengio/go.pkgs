// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"
	"sync"
	"sync/atomic"

	"cloudeng.io/errors"
	"cloudeng.io/file/content"
	"cloudeng.io/sync/errgroup"
)

// AsyncWrite represents a store for objects with asynchronous writes. Reads
// are synchronous. The caller must ensure that Finish is called to ensure
// that all writes have completed.
type Async struct {
	fs          content.FS
	written     int64
	read        int64
	concurrency int
	mu          sync.Mutex
	writer      *asyncWriter
	reader      *asyncReader
}

// EraseExisting deletes all contents of the store beneath root.
func (s *Async) EraseExisting(ctx context.Context, root string) error {
	return eraseExisting(ctx, s.fs, root)
}

func (s *Async) FS() content.FS {
	return s.fs
}

// Stats returns the number of objects read and written to the store
// since this instance was created.
func (s *Async) Stats() (read, written int64) {
	return atomic.LoadInt64(&s.read), atomic.LoadInt64(&s.written)
}

type asyncWriter struct {
	ch   chan writeRequest
	g    *errgroup.T
	errs *errors.M
}

type writeRequest struct {
	prefix, name string
	data         []byte
}

func NewAsync(fs content.FS, concurrency int) *Async {
	return &Async{
		fs:          fs,
		concurrency: concurrency,
	}
}

func (s *Async) newWriter(ctx context.Context) *asyncWriter {
	wr := &asyncWriter{
		ch:   make(chan writeRequest, s.concurrency),
		errs: &errors.M{},
		g:    &errgroup.T{},
	}
	for i := 0; i < s.concurrency; i++ {
		wr.g.Go(func() error {
			return wr.writer(ctx, s.fs, &s.written)
		})
	}
	return wr
}

func (wr *asyncWriter) writer(ctx context.Context, fs content.FS, counter *int64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-wr.ch:
			if !ok {
				return nil
			}
			err := write(ctx, fs, req.prefix, req.name, req.data, counter)
			wr.errs.Append(err)
		}
	}
}

// Write queues a write request for the specified prefix and name in the store.
// There is no guarantee that the write will have completed when this method
// returns. The error code returned is an indication that the write request
// was queued and will only ever context.Canceled if non-nil.
func (s *Async) Write(ctx context.Context, prefix, name string, data []byte) error {
	s.mu.Lock()
	if s.writer == nil {
		s.writer = s.newWriter(ctx)
	}
	ch := s.writer.ch
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- writeRequest{prefix, name, data}:
		return nil
	}
}

type asyncReader struct {
	ch   chan string
	g    *errgroup.T
	errs *errors.M
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Async) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error) {
	return read(ctx, s.fs, s.fs.Join(prefix, name), &s.read)
}

func (s *Async) newReader(ctx context.Context, prefix string,
	fn func(name string, typ content.Type, data []byte, err error) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case name, ok := <-s.rdCh:
			if !ok {
				return nil
			}
			typ, data, err := s.Read(ctx, prefix, name)
			s.errs.Append(err)
			if err := fn(name, typ, data, err); err != nil {
				return err
			}
		}
	}
}

func (s *Async) ReadAsync(ctx context.Context, prefix string, names []string,
	fn func(name string, typ content.Type, data []byte, err error) error) error {
	rdCh := make(chan string, s.concurrency)

	var errCh = make(chan error, 1)
	go func() {
		defer close(rdCh)
		for _, name := range names {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case s.rdCh <- name:
			}
		}
		errCh <- nil
	}()

	for i := 0; i < s.concurrency; i++ {
		s.g.Go(func() error {
			return s.reader(ctx, prefix, fn)
		})
	}

	s.errs.Append(<-errCh)
	s.errs.Append(s.g.Wait())
	return s.errs.Err()
}

// Finish waits for all queued writes to complete and returns any errors
// encountered during the writes.
func (s *Async) Finish() error {
	close(s.wrCh)
	errs := s.g.Wait()
	s.errs.Append(errs)
	return s.errs.Err()
}
