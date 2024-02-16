// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"
	"runtime"
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

func (wr *asyncWriter) finish() error {
	close(wr.ch)
	wr.errs.Append(wr.g.Wait())
	return wr.errs.Err()
}

// Finish waits for all queued writes to complete and returns any errors
// encountered during the writes.
func (s *Async) Finish() error {
	s.mu.Lock()
	if s.writer == nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	return s.writer.finish()
}

// ReadFunc is called by ReadAsync for each object read from the store.
// If the read operation returned an error it is passed to ReadFunc and if
// then returned by ReadFunc it will cause the entire ReadAsync operation
// to terminate and return an error.
type ReadFunc func(ctx context.Context, prefix, name string, typ content.Type, data []byte, err error) error

type asyncReader struct {
	ch           chan string
	readerDoneCh chan struct{}
	g            *errgroup.T
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Async) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error) {
	return read(ctx, s.fs, s.fs.Join(prefix, name), &s.read)
}

func (s *Async) newReader(ctx context.Context, prefix string, fn ReadFunc) *asyncReader {
	rd := &asyncReader{
		ch:           make(chan string, s.concurrency),
		g:            &errgroup.T{},
		readerDoneCh: make(chan struct{}, 1),
	}
	for i := 0; i < s.concurrency; i++ {
		rd.g.Go(func() error {
			return rd.reader(ctx, s.fs, prefix, &s.read, fn)
		})
	}
	return rd
}

func (rd *asyncReader) reader(ctx context.Context, fs content.FS, prefix string, counter *int64, fn ReadFunc) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case name, ok := <-rd.ch:
			if !ok {
				return nil
			}
			path := fs.Join(prefix, name)
			typ, data, err := read(ctx, fs, path, counter)
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

// ReadAsync retrieves the objects with the specified names from the store
// and calls fn for each object. The read operations are performed
// concurrently.
func (s *Async) ReadAsync(ctx context.Context, prefix string, names []string, fn ReadFunc) error {
	s.mu.Lock()
	if s.reader == nil {
		s.reader = s.newReader(ctx, prefix, fn)
	}
	s.mu.Unlock()

	var errs errors.M
	var errCh = make(chan error, 1)
	go func() {
		errCh <- s.reader.issueRequests(ctx, names)
	}()

	errs.Append(s.reader.g.Wait())
	close(s.reader.readerDoneCh)
	errs.Append(<-errCh)
	return errs.Err()
}
