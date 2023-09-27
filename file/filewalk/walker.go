// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package filewalk provides support for concurrent traversal of file system
// directories and files. It can traverse any filesytem that implements
// the Filesystem interface and is intended to be usable with cloud storage
// systems as AWS S3 or GCP's Cloud Storage. All compatible systems must
// implement some sort of hierarchical naming scheme, whether it be directory
// based (as per Unix/POSIX filesystems) or by convention (as per S3).
package filewalk

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/sync/errgroup"
)

// Walker implements the filesyste walk.
type Walker[T any] struct {
	fs       FS
	opts     options
	handler  Handler[T]
	errs     *errors.M
	nSyncOps int64
	tk       *timekeeper
}

// Option represents options accepted by Walker.
type Option func(o *options)

type options struct {
	concurrency    int
	scanSize       int
	reportCh       chan<- Status
	reportInterval time.Duration
	slowThreshold  time.Duration
}

// WithConcurrency can be used to change the degree of concurrency used. The
// default is to use all available CPUs.
func WithConcurrency(n int) Option {
	return func(o *options) {
		o.concurrency = n
	}
}

// WithScanSize sets the number of prefix/directory entries to be scanned
// in a single operation.
func WithScanSize(n int) Option {
	return func(o *options) {
		o.scanSize = n
	}
}

// Status is used to communicate the status of in-process Walk operation.
type Status struct {
	// SynchronousOps is the number of Scans that were performed synchronously
	// as a fallback when all available goroutines are already occupied.
	SynchronousScans int64

	// SlowPrefix is a prefix that took longer than a certain duration
	// to scan. ScanDuration is the time spent scanning that prefix to
	// date. A SlowPrefix may be reported as slow before it has completed
	// scanning.
	SlowPrefix   string
	ScanDuration time.Duration
}

// WithReporting can be used to enable reporting of slow and synchronous
// prefix/directory scans.
func WithReporting(ch chan<- Status, interval, slowThreshold time.Duration) Option {
	return func(o *options) {
		o.reportCh = ch
		o.reportInterval = interval
		o.slowThreshold = slowThreshold
	}
}

// New creates a new Walker instance.
func New[T any](fs FS, handler Handler[T], opts ...Option) *Walker[T] {
	w := &Walker[T]{fs: fs, handler: handler, errs: &errors.M{}}
	for _, fn := range opts {
		fn(&w.opts)
	}
	if w.opts.scanSize == 0 {
		w.opts.scanSize = 1000
	}
	w.tk = newTimekeeper(w.opts)
	return w
}

type LevelScanner interface {
	Scan(ctx context.Context, n int) bool
	Contents() Contents
	Err() error
}

// FS represents the interface that is implemeted for filesystems to
// be traversed/scanned.
type FS interface {
	file.FS

	// Readlink returns the contents of a symbolic link.
	Readlink(ctx context.Context, path string) (string, error)

	// Stat will follow symlinks.
	Stat(ctx context.Context, path string) (file.Info, error)

	// LStat will not follow symlinks.
	LStat(ctx context.Context, path string) (file.Info, error)

	LevelScanner(path string) LevelScanner

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// IsPermissionError returns true if the specified error, as returned
	// by the filesystem's implementation, is a result of a permissions error.
	IsPermissionError(err error) bool

	// IsNotExist returns true if the specified error, as returned by the
	// filesystem's implementation, is a result of the object not existing.
	IsNotExist(err error) bool
}

// Error implements error and provides additional detail on the error
// encountered.
type Error struct {
	Path string
	Op   string
	Err  error
}

// Error implements error.
func (e *Error) Error() string {
	return "[" + e.Path + ": " + e.Op + "] " + e.Err.Error()
}

// recordError will record the specified error if it is not nil; ie.
// its safe to call it with a nil error.
func (w *Walker[T]) recordError(path, op string, err error) {
	if err == nil {
		return
	}
	w.errs.Append(&Error{path, op, err})
}

func (w *Walker[T]) listLevel(ctx context.Context, state *T, path string, unchanged bool, info file.Info) []Entry {

	w.tk.add(path)
	defer w.tk.rm(path)

	var nextLevel []Entry
	sc := w.fs.LevelScanner(path)
	for sc.Scan(ctx, w.opts.scanSize) {
		children, err := w.handler.Contents(ctx, state, path, unchanged, info, sc.Contents())
		if err != nil {
			w.recordError(path, "handler.Contents", err)
			return nil
		}
		nextLevel = append(nextLevel, children...)
	}

	if err := sc.Err(); err != nil {
		if _, err := w.handler.Contents(ctx, state, path, unchanged, info, Contents{Err: err}); err != nil {
			w.recordError(path, "handler.Contents", err)
			return nil
		}
	}

	return nextLevel
}

type Handler[T any] interface {

	// PrefixFunc is called to determine if a given level in the filesystem hiearchy
	// should be further examined or traversed. The file.Info is obtained via a call
	// to LStat and hence will refer to a symlink itself if the prefix is a symlink.
	// If stop is true then traversal stops at this point. If a list of children
	// is returned then this list is traversed directly rather than obtaining
	// the children from the filesystem.
	// This allows for both exclusions and incremental processing in conjunction
	// with a database to be implemented.

	// ContentsFunc is called multiple times to consume the results of scanning
	// a single level in the filesystem hierarchy. Each call contains at most
	// the number of items allowed for by the WithScanSize option.
	// ionfo is the same as that passed to PrefixFunc and unchanged is the value
	// returned by PrefixFunc.
	// Errors, such as failing to access the prefix, are delivered over the channel.

	Prefix(ctx context.Context, state *T, prefix string, info file.Info, err error) (stop, children []Entry, returnErr error)
	Contents(ctx context.Context, state *T, contents Contents) ([]Entry, error)
	Done(ctx context.Context, state *T) error
}

// Walk traverses the hierarchies specified by each of the roots calling
// prefixFn and entriesFn as it goes. prefixFn will always be called
// before entriesFn for the same prefix, but no other ordering guarantees
// are provided.
func (w *Walker[T]) Walk(ctx context.Context, roots ...string) error {
	rootCtx := ctx
	listers, _ := errgroup.WithContext(rootCtx)
	if w.opts.concurrency <= 0 {
		w.opts.concurrency = runtime.GOMAXPROCS(-1)
	}

	listers = errgroup.WithConcurrency(listers, w.opts.concurrency)

	walkers, _ := errgroup.WithContext(rootCtx)
	walkers = errgroup.WithConcurrency(walkers, w.opts.concurrency)

	// create and prime the concurrency limiter for walking directories.
	walkerLimitCh := make(chan struct{}, w.opts.concurrency*2)
	for i := 0; i < cap(walkerLimitCh); i++ {
		walkerLimitCh <- struct{}{}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var reportingDoneCh, reportingStopCh chan struct{}
	if w.opts.reportCh != nil {
		reportingDoneCh, reportingStopCh = make(chan struct{}), make(chan struct{})
		go func() {
			for {
				if w.report(rootCtx, reportingStopCh) {
					close(w.opts.reportCh)
					close(reportingDoneCh)
					return
				}
				time.Sleep(w.opts.reportInterval)
			}
		}()
	}

	for _, root := range roots {
		root := root
		walkers.Go(func() error {
			<-walkerLimitCh
			w.walker(ctx, root, walkerLimitCh)
			return nil
		})
	}

	go func() {
		w.errs.Append(listers.Wait())
		wg.Done()
	}()

	go func() {
		w.errs.Append(walkers.Wait())
		wg.Done()
	}()

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-rootCtx.Done():
		w.errs.Append(rootCtx.Err())
	case <-waitCh:
	}

	if w.opts.reportCh != nil {
		close(reportingStopCh)
		select {
		case <-rootCtx.Done():
			w.errs.Append(rootCtx.Err())
		case <-reportingDoneCh:
		}
	}

	return w.errs.Err()
}

func (w *Walker[T]) walkChildren(ctx context.Context, path string, children []Entry, limitCh chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(len(children))
	for _, child := range children {
		child := child
		select {
		case <-limitCh:
		case <-ctx.Done():
			return
		default:
			// no concurreny is available fallback to sync.
			atomic.AddInt64(&w.nSyncOps, 1)
			p := w.fs.Join(path, child.Name)

			w.walker(ctx, p, limitCh)
			wg.Done()
			continue
		}
		go func() {
			w.walker(ctx, w.fs.Join(path, child.Name), limitCh)
			wg.Done()
			limitCh <- struct{}{}
		}()
	}
	wg.Wait()
}

func (w *Walker[T]) walker(ctx context.Context, path string, limitCh chan struct{}) {
	select {
	default:
	case <-ctx.Done():
		return
	}
	var state T
	info, err := w.fs.LStat(ctx, path)

	stop, children, err := w.handler.Prefix(ctx, &state, path, info, err)
	w.recordError(path, "handler.Prefix", err)
	if stop {
		return
	}
	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
		if err := w.handler.Done(ctx, &state, path, info); err != nil {
			w.recordError(path, "handler.Done", err)
		}
		return
	}

	children = w.listLevel(ctx, &state, path, info)

	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
	}
	if err := w.handler.Done(ctx, &state, path, info); err != nil {
		w.recordError(path, "handler.Done", err)
	}
}

func (w *Walker[T]) report(ctx context.Context, stopCh <-chan struct{}) bool {
	select {
	case <-ctx.Done():
		return true
	case <-stopCh:
		return true
	case w.opts.reportCh <- Status{
		SynchronousScans: atomic.LoadInt64(&w.nSyncOps),
	}:
	default:
	}
	w.tk.report()
	return false
}

func newTimekeeper(opts options) *timekeeper {
	if opts.reportCh == nil {
		return &timekeeper{}
	}
	return &timekeeper{
		prefixes: make(map[string]time.Time, opts.concurrency),
		ch:       opts.reportCh,
		slow:     opts.slowThreshold,
	}
}

type timekeeper struct {
	sync.Mutex
	prefixes map[string]time.Time
	ch       chan<- Status
	slow     time.Duration
}

func (tk *timekeeper) add(prefix string) {
	if tk.slow == 0 {
		return
	}
	tk.Lock()
	defer tk.Unlock()
	tk.prefixes[prefix] = time.Now()
}

func (tk *timekeeper) rm(prefix string) {
	if tk.slow == 0 {
		return
	}
	tk.Lock()
	defer tk.Unlock()
	delete(tk.prefixes, prefix)
}

func (tk *timekeeper) report() {
	if tk.slow == 0 {
		return
	}
	tk.Lock()
	defer tk.Unlock()
	for prefix, t := range tk.prefixes {
		if since := time.Since(t); since > tk.slow {
			select {
			case tk.ch <- Status{SlowPrefix: prefix, ScanDuration: since}:
			default:
			}
		}
	}
}
