// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"
	"runtime"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file/content"
	"cloudeng.io/sync/errgroup"
)

// AsyncWrite represents a store for objects with asynchronous writes. Reads
// are synchronous. The caller must ensure that Finish is called to ensure
// that all writes have completed.
type Async struct {
	fs          content.FS
	concurrency int
	mu          sync.Mutex
	writer      *asyncWriter
}

// EraseExisting deletes all contents of the store beneath root.
func (s *Async) EraseExisting(ctx context.Context, root string) error {
	return eraseExisting(ctx, s.fs, root)
}

func (s *Async) FS() content.FS {
	return s.fs
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

// NewAsync returns a new instance of Async with the specified concurrency.
// If concurrency is less than or equal to zero, the number of CPUs is used.
func NewAsync(fs content.FS, concurrency int) *Async {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
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
			return wr.writer(ctx, s.fs)
		})
	}
	return wr
}

func (wr *asyncWriter) writer(ctx context.Context, fs content.FS) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-wr.ch:
			if !ok {
				return nil
			}
			err := write(ctx, fs, req.prefix, req.name, req.data)
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

func (wr *asyncWriter) finish() error {
	close(wr.ch)
	wr.errs.Append(wr.g.Wait())
	return wr.errs.Err()
}

// Finish waits for all queued writes to complete and returns any errors
// encountered during the writes.
func (s *Async) Finish(context.Context) error {
	s.mu.Lock()
	if s.writer == nil {
		s.mu.Unlock()
		return nil
	}
	writer := s.writer
	s.writer = nil
	s.mu.Unlock()
	return writer.finish()
}

type asyncReader struct {
	ch           chan string
	readerDoneCh chan struct{}
	g            *errgroup.T
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Async) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error) {
	return read(ctx, s.fs, s.fs.Join(prefix, name))
}

func (s *Async) newReader(ctx context.Context, prefix string, fn ReadFunc) *asyncReader {
	rd := &asyncReader{
		ch:           make(chan string, s.concurrency),
		g:            &errgroup.T{},
		readerDoneCh: make(chan struct{}, 1),
	}
	for i := 0; i < s.concurrency; i++ {
		rd.g.Go(func() error {
			return rd.reader(ctx, s.fs, prefix, fn)
		})
	}
	return rd
}

func (rd *asyncReader) reader(ctx context.Context, fs content.FS, prefix string, fn ReadFunc) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case name, ok := <-rd.ch:
			if !ok {
				return nil
			}
			path := fs.Join(prefix, name)
			typ, data, err := read(ctx, fs, path)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err := fn(ctx, prefix, name, typ, data, err); err != nil {
				return err
			}
		}
	}
}

func (rd *asyncReader) issueRequests(ctx context.Context, names []string) error {
	defer close(rd.ch)
	for _, name := range names {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rd.readerDoneCh:
			break
		case rd.ch <- name:
		}
	}
	return nil
}

// ReadV retrieves the objects with the specified names from the store
// and calls fn for each object. The read operations are performed
// concurrently.
func (s *Async) ReadV(ctx context.Context, prefix string, names []string, fn ReadFunc) error {
	reader := s.newReader(ctx, prefix, fn)

	var errs errors.M
	var errCh = make(chan error, 1)
	go func() {
		errCh <- reader.issueRequests(ctx, names)
	}()

	errs.Append(reader.g.Wait())
	close(reader.readerDoneCh)
	errs.Append(<-errCh)
	return errs.Err()
}
