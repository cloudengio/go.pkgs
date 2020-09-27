// Package filewalk provides support for efficient, concurrent traversing
// of file system directories.
package filewalk

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

type Visitor func(ctx context.Context, path string, info os.FileInfo, err error) error

type Walker struct {
	scanner Scanner
	opts    options
	errs    *errors.M
}

type Option func(o *options)

type options struct {
	concurrency int
	scanSize    int
}

func Concurrency(n int) Option {
	return func(o *options) {
		o.concurrency = n
	}
}

func ScanSize(n int) Option {
	return func(o *options) {
		o.scanSize = n
	}
}

func New(scanner Scanner, opts ...Option) *Walker {
	w := &Walker{scanner: scanner, errs: &errors.M{}}
	for _, fn := range opts {
		fn(&w.opts)
	}
	return w
}

type ScanRecord struct {
	Entries []os.FileInfo
	Err     error
}

type Scanner interface {
	List(ctx context.Context, path string, n int, ch chan<- ScanRecord) error
	Join(components ...string) string
}

type walker struct {
	scanner Scanner
	options options
}

func (w *Walker) Walk(ctx context.Context, roots []string, visitor Visitor) error {
	eg := errgroup.WithConcurrency(&errgroup.T{}, w.opts.concurrency)
	eg, ctx = errgroup.WithContext(ctx)
	size := len(roots)
	if size > w.opts.concurrency {
		size = w.opts.concurrency
	}
	ch := make(chan ScanRecord, size)
	for _, root := range roots {
		eg.Go(func() error {
			w.scanner.List(ctx, root, w.opts.scanSize, ch)
			return nil
		})
		eg.Go(func() error {
			// read on chan, call Visitor.
		}
	}

	// QUESTION: how to handle/bundle directories.
	// return from visitor to indicate if you should walk further.

	err := eg.Wait()
}

type local struct{}

func (l *local) List(ctx context.Context, path string, n int, ch chan<- ScanRecord) {
	f, err := os.Open(path)
	if err != nil {
		ch <- ScanRecord{
			Err: err,
		}
	}
	for {
		fi, err := f.Readdir(n)
		ch <- ScanRecord{
			Entries: fi,
			Err:     err,
		}
		if err == io.EOF {
			return nil
		}
	}
}

func (l *local) Join(components ...string) string {
	return filepath.Join(components...)
}

func LocalScanner() Scanner {
	return &local{}
}
