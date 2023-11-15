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
	concurrentScans int
	scanSize        int
	reportCh        chan<- Status
	reportInterval  time.Duration
	slowThreshold   time.Duration
}

// WithConcurrentScans can be used to change the number of prefixes/directories
// that can be scanned concurrently.
// The default is DefaultConcurrentScans.
func WithConcurrentScans(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.concurrentScans = n
		}
	}
}

// WithScanSize sets the number of prefix/directory entries to be scanned
// in a single operation.
// The default is DefaultScanSize.
func WithScanSize(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.scanSize = n
		}
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

var (
	// DefaultScansize is the default ScanSize used when the WithScanSize
	// option is not supplied.
	DefaultScanSize = 1000
	// DefaultConcurrentScans is the default number of prefixes/directories
	// that will be scanned concurrently when the WithConcurrencyOption is
	// is not supplied.
	DefaultConcurrentScans = 100
)

// New creates a new Walker instance.
func New[T any](fs FS, handler Handler[T], opts ...Option) *Walker[T] {
	w := &Walker[T]{fs: fs, handler: handler, errs: &errors.M{}}
	w.opts.concurrentScans = DefaultScanSize
	w.opts.scanSize = DefaultScanSize
	for _, fn := range opts {
		fn(&w.opts)
	}
	w.tk = newTimekeeper(w.opts)
	return w
}

type LevelScanner interface {
	Scan(ctx context.Context, n int) bool
	Contents() []Entry
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

	// Lstat will not follow symlinks.
	Lstat(ctx context.Context, path string) (file.Info, error)

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
	Err  error
}

// Error implements error.
func (e Error) Error() string {
	return "[" + e.Path + ": " + e.Err.Error()
}

// Is implements errors.Is.
func (e Error) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// Unwrap implements errors.Unwrap.
func (e Error) Unwrap() error {
	return e.Err
}

// As implements errors.As.
func (e *Error) As(target interface{}) bool {
	if t, ok := target.(*Error); ok {
		*t = *e
		return true
	}
	return errors.As(e.Err, target)
}

func (w *Walker[T]) processLevel(ctx context.Context, state *T, path string) ([]file.Info, error) {

	w.tk.add(path)
	defer w.tk.rm(path)

	var nextLevel []file.Info

	sc := w.fs.LevelScanner(path)
	for sc.Scan(ctx, w.opts.scanSize) {
		children, err := w.handler.Contents(ctx, state, path, sc.Contents())
		if err != nil {
			return nil, err
		}
		nextLevel = append(nextLevel, children...)
	}
	return nextLevel, sc.Err()
}

// Handler is implemented by clients of Walker to process the results of
// walking a filesystem hierarchy. The type parameter is used to instantiate a
// state variable that is passed to each of the methods.
type Handler[T any] interface {

	// Prefix is called to determine if a given level in the filesystem hiearchy
	// should be further examined or traversed. The file.Info is obtained via a call
	// to Lstat and hence will refer to a symlink itself if the prefix is a symlink.
	// If stop is true then traversal stops at this point. If a list of Entry's
	// is returned then this list is traversed directly rather than obtaining
	// the children from the filesystem. This allows for both exclusions and
	// incremental processing in conjunction with a database to be implemented.
	// Any returned is recorded, but traversal will continue unless stop is set.
	Prefix(ctx context.Context, state *T, prefix string, info file.Info, err error) (stop bool, children file.InfoList, returnErr error)

	// Contents is called, multiple times, to process the contents of a single
	// level in the filesystem hierarchy. Each such call contains at most the
	// number of items allowed for by the WithScanSize option. Note that
	// errors encountered whilst scanning the filesystem result in calls to
	// Done with the error encountered.
	Contents(ctx context.Context, state *T, prefix string, contents []Entry) (file.InfoList, error)

	// Done is called once calls to Contents have been made or if Prefix returned
	// an error. Done will always be called if Prefix did not return true for stop.
	// Errors returned by Done are recorded and returned by the Walk method.
	Done(ctx context.Context, state *T, prefix string, err error) error
}

// Walk traverses the hierarchies specified by each of the roots calling
// prefixFn and entriesFn as it goes. prefixFn will always be called
// before entriesFn for the same prefix, but no other ordering guarantees
// are provided.
func (w *Walker[T]) Walk(ctx context.Context, roots ...string) error {
	rootCtx := ctx
	listers, _ := errgroup.WithContext(rootCtx)
	if w.opts.concurrentScans <= 0 {
		w.opts.concurrentScans = runtime.GOMAXPROCS(-1)
	}

	listers = errgroup.WithConcurrency(listers, w.opts.concurrentScans)

	walkers, _ := errgroup.WithContext(rootCtx)
	walkers = errgroup.WithConcurrency(walkers, w.opts.concurrentScans)

	// create and prime the concurrentScans limiter for walking directories.
	walkerLimitCh := make(chan struct{}, w.opts.concurrentScans*2)
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
		rootInfo, rootErr := w.fs.Lstat(ctx, root)
		walkers.Go(func() error {
			<-walkerLimitCh
			w.walker(ctx, root, rootInfo, rootErr, walkerLimitCh)
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
		<-waitCh
	case <-waitCh:
	}

	if w.opts.reportCh != nil {
		close(reportingStopCh)
		select {
		case <-rootCtx.Done():
			w.errs.Append(rootCtx.Err())
			<-reportingDoneCh
		case <-reportingDoneCh:
		}
	}

	return w.errs.Err()
}

func (w *Walker[T]) walkChildren(ctx context.Context, path string, children []file.Info, limitCh chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(len(children))
	for _, child := range children {
		child := child
		select {
		case <-limitCh:
		default:
			// no concurreny is available fallback to sync.
			atomic.AddInt64(&w.nSyncOps, 1)
			p := w.fs.Join(path, child.Name())
			w.walker(ctx, p, child, nil, limitCh)
			wg.Done()
			continue
		}
		go func() {
			w.walker(ctx, w.fs.Join(path, child.Name()), child, nil, limitCh)
			wg.Done()
			limitCh <- struct{}{}
		}()
	}
	wg.Wait()
}

func (w *Walker[T]) handleDone(ctx context.Context, state *T, path string, err error) {
	err = w.handler.Done(ctx, state, path, err)
	if err == nil {
		return
	}
	w.errs.Append(&Error{path, err})
}

func (w *Walker[T]) walker(ctx context.Context, path string, info file.Info, err error, limitCh chan struct{}) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	var state T
	stop, children, err := w.handler.Prefix(ctx, &state, path, info, err)
	if stop {
		return
	}
	if err != nil {
		w.handleDone(ctx, &state, path, err)
		return
	}
	if len(children) == 0 {
		children, err = w.processLevel(ctx, &state, path)
		if err != nil {
			w.handleDone(ctx, &state, path, err)
			return
		}
	}
	w.walkChildren(ctx, path, children, limitCh)
	w.handleDone(ctx, &state, path, nil)
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
		prefixes: make(map[string]time.Time, opts.concurrentScans),
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

type Configuration struct {
	ConcurrentScans int
	ScanSize        int
	ReportInterval  time.Duration
}

func (w *Walker[T]) Configuration() Configuration {
	return Configuration{
		ConcurrentScans: w.opts.concurrentScans,
		ScanSize:        w.opts.scanSize,
		ReportInterval:  w.opts.reportInterval,
	}
}
