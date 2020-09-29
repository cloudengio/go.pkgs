// Package filewalk provides support for concurrent traversal of file system
// directories and files. It can traverse any filesytem that implements
// the Scanner interface and is intended to be usable with cloud storage
// systems as AWS S3 or GCP's Cloud Storage. All such systems must implement
// some sort of hierarchical naming scheme, whether it be directory based
// (as per Unix/POSIX filesystems) or by convention (as per S3).
package filewalk

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

// Contents represents the contents of the filesystem at the level represented
// by Path.
type Contents struct {
	Path     string // The name of the level being scanned.
	Children []Info `json:,omitempty` // Info on each of the next levels in the hierarchy.
	Files    []Info `json:,omitempty` // Info for the files at this level.
	Err      error  `json:,omitempty` // Non-nil if an error occurred.
}

// ContentsFunc is the type of the function that is called to consume the results
// of scanning a single level in the filesystem hierarchy. Walker.Walk will
// call this function and deliver all results for that level via the supplied
// channel; that channel is closed when there is no more data. Errors, such
// as failing to access the prefix, are delivered over the channel.
type ContentsFunc func(ctx context.Context, prefix string, ch <-chan Contents) error

// PrefixFunc is the type of the function that is called to determine if a given
// level in the filesystem hiearchy should be further examined. If the skip
// result is true then traversal will stop at this level.
type PrefixFunc func(ctx context.Context, prefix string, info Info, err error) (skip bool, returnErr error)

// Walker implements the filesyste walk.
type Walker struct {
	scanner    Scanner
	opts       options
	listers    *errgroup.T
	contentsFn ContentsFunc
	prefixFn   PrefixFunc
	errs       *errors.M
}

// Option represents options accepted by Walker.
type Option func(o *options)

type options struct {
	concurrency int
	scanSize    int
	chanSize    int
}

// Concurreny can be used to change the degree of concurrency used. The
// default is to use all available CPUs.
func Concurrency(n int) Option {
	return func(o *options) {
		o.concurrency = n
	}
}

// ScanSize can be used to control the number of items retrieved per
// filesystem scan, it is used to set the scanSize paramter for Scanner.List.
// It defaults to 1000.
func ScanSize(n int) Option {
	return func(o *options) {
		o.scanSize = n
	}
}

// ChanSize can be used to set the size of the channel used to send results
// to ResultsFunc. It defaults to being unbuffered.
func ChanSize(n int) Option {
	return func(o *options) {
		o.chanSize = n
	}
}

// New creates a new Walker instance.
func New(scanner Scanner, opts ...Option) *Walker {
	w := &Walker{scanner: scanner, errs: &errors.M{}}
	w.opts.chanSize = 0
	w.opts.scanSize = 1000
	for _, fn := range opts {
		fn(&w.opts)
	}
	return w
}

// Info represents the information that can be retrieved for a filesystem
// entry.
type Info interface {
	Name() string       // base name of the file
	Size() int64        // length in bytes
	ModTime() time.Time // modification time
	IsPrefix() bool     // true if this Info refers to a directory or prefix
	IsLink() bool       // true if this Info refers to a link, e.g. a symbolic link
	Sys() interface{}   // underlying data source (can return nil)
}

// Scanner represents the interface that is implemeted for filesystems to
// be scanned.
type Scanner interface {
	// Stat obtains Info for the specified path.
	Stat(ctx context.Context, path string) (Info, error)
	// Join is used
	Join(components ...string) string
	List(ctx context.Context, path string, scanSize int, ch chan<- Contents)
}

type Error struct {
	Path string
	Op   string
	Err  error
}

func (e *Error) Error() string {
	return e.Path + ": " + e.Op + ": " + e.Err.Error()
}

func (w *Walker) recordError(path, op string, err error) error {
	if err == nil {
		return nil
	}
	w.errs.Append(&Error{path, op, err})
	return err
}

func (w *Walker) listLevel(ctx context.Context, path string, info Info) []Info {
	var wg sync.WaitGroup
	wg.Add(3)

	ch := make(chan Contents, w.opts.concurrency)
	rch := make(chan Contents, w.opts.chanSize)

	go func(path string) {
		w.scanner.List(ctx, path, w.opts.scanSize, ch)
		close(ch)
		wg.Done()
	}(path)

	var children []Info

	go func() {
		defer close(rch)
		defer wg.Done()
		for results := range ch {
			select {
			case <-ctx.Done():
				w.recordError(path, "listLevel", ctx.Err())
				return
			default:
			}
			children = append(children, results.Children...)
			rch <- results
		}
	}()

	go func(path string) {
		err := w.contentsFn(ctx, path, rch)
		w.recordError(path, "fileFunc", err)
		wg.Done()
	}(path)

	wg.Wait()

	sort.Slice(children, func(i, j int) bool {
		if children[i].Name() == children[j].Name() {
			return children[i].Size() >= children[j].Size()
		}
		return children[i].Name() < children[j].Name()
	})
	return children
}

type listRequest struct {
	path    string
	info    Info
	childCh chan<- []Info
}

func (w *Walker) lister(ctx context.Context, workCh chan listRequest) {
	for work := range workCh {
		children := w.listLevel(ctx, work.path, work.info)
		select {
		case <-ctx.Done():
			w.recordError(work.path, "ctx.Done()", ctx.Err())
		default:
		}
		work.childCh <- children
		close(work.childCh)
	}
	return
}

func (w *Walker) Walk(ctx context.Context, prefixFn PrefixFunc, contentsFn ContentsFunc, roots ...string) error {
	rootCtx := ctx
	listers, ctx := errgroup.WithContext(rootCtx)
	if w.opts.concurrency <= 0 {
		w.opts.concurrency = runtime.GOMAXPROCS(-1)
	}
	listers = errgroup.WithConcurrency(listers, w.opts.concurrency)

	walkers, ctx := errgroup.WithContext(rootCtx)
	walkers = errgroup.WithConcurrency(walkers, w.opts.concurrency)

	w.prefixFn = prefixFn
	w.contentsFn = contentsFn

	requestCh := make(chan listRequest, w.opts.concurrency*2)

	for i := 0; i < w.opts.concurrency; i++ {
		listers.Go(func() error {
			w.lister(ctx, requestCh)
			return nil
		})
	}

	for _, root := range roots {
		root := root
		walkers.Go(func() error {
			w.walker(ctx, root, requestCh)
			return nil
		})
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		w.errs.Append(listers.Wait())
		wg.Done()
	}()

	go func() {
		w.errs.Append(walkers.Wait())
		close(requestCh)
		wg.Done()
	}()

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-rootCtx.Done():
		w.recordError("", "ctx.Done()", rootCtx.Err())
	case <-waitCh:
	}
	return w.errs.Err()
}

func (w *Walker) start(ctx context.Context, path string, requestCh chan<- listRequest) {
	info, err := w.scanner.Stat(ctx, path)
	if err == nil && !info.IsPrefix() {

	}
	w.walker(ctx, path, requestCh)
}

func (w *Walker) walker(ctx context.Context, path string, requestCh chan<- listRequest) {
	info, err := w.scanner.Stat(ctx, path)
	if err != nil {
		info = nil
	}
	// Only call prefixFn if the stat fails and there is no prospect
	// of listing its contents.
	skip, err := w.prefixFn(ctx, path, info, err)
	w.recordError(path, "stat", err)
	if skip {
		return
	}
	ch := make(chan []Info, 1)
	requestCh <- listRequest{path, info, ch}
	select {
	case children := <-ch:
		for _, child := range children {
			w.walker(ctx, w.scanner.Join(path, child.Name()), requestCh)
		}
	case <-ctx.Done():
		w.recordError(path, "ctx.Done()", ctx.Err())
	}
}
