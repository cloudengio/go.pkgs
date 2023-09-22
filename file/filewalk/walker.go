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

// Contents represents the contents of the filesystem at the level represented
// by Path.
type Contents struct {
	Path     string      `json:"p,omitempty"` // The name of the level being scanned.
	Children []file.Info `json:"c,omitempty"` // Info on each of the next levels in the hierarchy.
	Files    []file.Info `json:"f,omitempty"` // Info for the files at this level.
	Err      error       `json:"e,omitempty"` // Non-nil if an error occurred.
}

// Walker implements the filesyste walk.
type Walker struct {
	fs                            Filesystem
	opts                          options
	contentsFn                    ContentsFunc
	prefixFn                      PrefixFunc
	errs                          *errors.M
	nSyncOps                      int64
	prefixStarted, prefixFinished int64
	files                         int64
	tk                            *timekeeper
}

// Option represents options accepted by Walker.
type Option func(o *options)

type options struct {
	concurrency    int
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

func WithReporting(ch chan<- Status, interval, slowThreshold time.Duration) Option {
	return func(o *options) {
		o.reportCh = ch
		o.slowThreshold = slowThreshold
		o.reportInterval = interval
	}
}

// New creates a new Walker instance.
func New(filesystem Filesystem, opts ...Option) *Walker {
	w := &Walker{fs: filesystem, errs: &errors.M{}}
	for _, fn := range opts {
		fn(&w.opts)
	}
	w.tk = newTimekeeper(w.opts)
	return w
}

type FilesystemStats struct {
	NumList, NumStat, NumFiles, NumPrefixes int64
}

// Filesystem represents the interface that is implemeted for filesystems to
// be traversed/scanned.
type Filesystem interface {
	// Stat obtains Info for the specified path.
	Stat(ctx context.Context, path string) (file.Info, error)

	// Stats reports statistics on the filesystem's invocations and contents.
	Stats() FilesystemStats

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// List will send all of the contents of path over the supplied channel.
	List(ctx context.Context, path string, dirsOnly bool, ch chan<- Contents)

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
func (w *Walker) recordError(path, op string, err error) {
	if err == nil {
		return
	}
	w.errs.Append(&Error{path, op, err})
}

func (w *Walker) listLevel(ctx context.Context, path string, unchanged bool, info file.Info) file.InfoList {
	ch := make(chan Contents, w.opts.concurrency)

	go func(path string) {
		w.tk.add(path)
		w.fs.List(ctx, path, unchanged, ch)
		close(ch)
		w.tk.rm(path)
	}(path)

	children, err := w.contentsFn(ctx, path, unchanged, info, ch)

	if err != nil {
		w.recordError(path, "fileFunc", err)
		return nil
	}

	select {
	case <-ctx.Done():
		return nil
	case <-ch:
	}
	return children
}

// ContentsFunc is the type of the function that is called to consume the results
// of scanning a single level in the filesystem hierarchy. It should read
// the contents of the supplied channel until that channel is closed. Note
// unchanged is set to the value returned by the preceeding call to PrefixFunc
// and indicates that no file content should be expected, only prefixes/directories.
// Errors, such as failing to access the prefix, are delivered over the channel.
type ContentsFunc func(ctx context.Context, prefix string, unchanged bool, info file.Info, ch <-chan Contents) (file.InfoList, error)

// PrefixFunc is the type of the function that is called to determine if a given
// level in the filesystem hiearchy should be further examined or traversed.
// If stop is true then traversal stops at this point. If a list of children
// is returned then this list is traversed directly rather than obtaining
// the children from the filesystem. If unchanged is true, then traversal
// continues, but files are ignored at the next level.
// This allows for both
// exclusions and incremental processing in conjunction with a database to
// be implemented.
type PrefixFunc func(ctx context.Context, prefix string, info file.Info, err error) (stop, unchanged bool, children file.InfoList, returnErr error)

// Walk traverses the hierarchies specified by each of the roots calling
// prefixFn and contentsFn as it goes. prefixFn will always be called
// before contentsFn for the same prefix, but no other ordering guarantees
// are provided.
func (w *Walker) Walk(ctx context.Context, prefixFn PrefixFunc, contentsFn ContentsFunc, roots ...string) error {
	rootCtx := ctx
	listers, _ := errgroup.WithContext(rootCtx)
	if w.opts.concurrency <= 0 {
		w.opts.concurrency = runtime.GOMAXPROCS(-1)
	}

	listers = errgroup.WithConcurrency(listers, w.opts.concurrency)

	walkers, _ := errgroup.WithContext(rootCtx)
	walkers = errgroup.WithConcurrency(walkers, w.opts.concurrency)

	w.prefixFn = prefixFn
	w.contentsFn = contentsFn

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

func (w *Walker) walkChildren(ctx context.Context, path string, children []file.Info, limitCh chan struct{}) {
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
			p := w.fs.Join(path, child.Name())

			w.walker(ctx, p, limitCh)
			wg.Done()
			continue
		}
		go func() {
			w.walker(ctx, w.fs.Join(path, child.Name()), limitCh)
			wg.Done()
			limitCh <- struct{}{}
		}()
	}
	wg.Wait()

}

func (w *Walker) walker(ctx context.Context, path string, limitCh chan struct{}) {
	select {
	default:
	case <-ctx.Done():
		return
	}
	info, err := w.fs.Stat(ctx, path)
	stop, unchanged, children, err := w.prefixFn(ctx, path, info, err)
	w.recordError(path, "stat", err)
	if stop {
		return
	}
	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
		return
	}
	children = w.listLevel(ctx, path, unchanged, info)
	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
	}
}

func (w *Walker) report(ctx context.Context, stopCh <-chan struct{}) bool {
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
	return w.tk.report()
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

func (tk *timekeeper) report() bool {
	if tk.slow == 0 {
		return false
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
	return false
}
