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
	"expvar"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

var listingVar = expvar.NewMap("cloudeng.io/file/filewalk.listing")
var walkingVar = expvar.NewMap("cloudeng.io/file/filewalk.walking")
var syncVar = expvar.NewMap("cloudeng.io/file/filewalk.sync")

// Contents represents the contents of the filesystem at the level represented
// by Path.
type Contents struct {
	Path     string `json:"p,omitempty"` // The name of the level being scanned.
	Children []Info `json:"c,omitempty"` // Info on each of the next levels in the hierarchy.
	Files    []Info `json:"f,omitempty"` // Info for the files at this level.
	Err      error  `json:"e,omitempty"` // Non-nil if an error occurred.
}

// Walker implements the filesyste walk.
type Walker struct {
	fs         Filesystem
	opts       options
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

// ChanSize can be used to set the size of the channel used to send results
// to ResultsFunc. It defaults to being unbuffered.
func ChanSize(n int) Option {
	return func(o *options) {
		o.chanSize = n
	}
}

// New creates a new Walker instance.
func New(filesystem Filesystem, opts ...Option) *Walker {
	w := &Walker{fs: filesystem, errs: &errors.M{}}
	w.opts.chanSize = 1000
	w.opts.scanSize = 1000
	for _, fn := range opts {
		fn(&w.opts)
	}
	return w
}

// FileMode represents meta data about a single file, including its
// permissions. Not all underlying filesystems may support the full
// set of UNIX-style permissions.
type FileMode uint32

const (
	ModePrefix FileMode = FileMode(os.ModeDir)
	ModeLink   FileMode = FileMode(os.ModeSymlink)
	ModePerm   FileMode = FileMode(os.ModePerm)
)

// String implements jsonString.
func (fm FileMode) String() string {
	return os.FileMode(fm).String()
}

// Info represents the information that can be retrieved for a single
// file or prefix.
type Info struct {
	Name    string      // base name of the file
	UserID  string      // user id as returned by the underlying system
	GroupID string      // group id as returned by the underlying system
	Size    int64       // length in bytes
	ModTime time.Time   // modification time
	Mode    FileMode    // permissions, directory or link.
	sys     interface{} // underlying data source (can return nil)
}

// Sys returns the underlying, if available, data source.
func (i Info) Sys() interface{} {
	return i.sys
}

// IsPrefix returns true for a prefix.
func (i Info) IsPrefix() bool {
	return (i.Mode & ModePrefix) == ModePrefix
}

// IsLink returns true for a symbolic or other form of link.
func (i Info) IsLink() bool {
	return (i.Mode & ModeLink) == ModeLink
}

// Perms returns UNIX-style permissions.
func (i Info) Perms() FileMode {
	return (i.Mode & ModePerm)
}

// Filesystem represents the interface that is implemeted for filesystems to
// be traversed/scanned.
type Filesystem interface {
	// Stat obtains Info for the specified path.
	Stat(ctx context.Context, path string) (Info, error)

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// List will send all of the contents of path over the supplied channel.
	List(ctx context.Context, path string, ch chan<- Contents)

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
func (w *Walker) recordError(path, op string, err error) error {
	if err == nil {
		return nil
	}
	w.errs.Append(&Error{path, op, err})
	return err
}

func (w *Walker) listLevel(ctx context.Context, idx string, path string, info *Info) []Info {
	listingVar.Set(idx, jsonString(path))
	defer listingVar.Delete(idx)
	ch := make(chan Contents, w.opts.concurrency)

	go func(path string) {
		w.fs.List(ctx, path, ch)
		close(ch)
	}(path)

	children, err := w.contentsFn(ctx, path, info, ch)

	if err != nil {
		w.recordError(path, "fileFunc", err)
		return nil
	}

	select {
	case <-ctx.Done():
		return nil
	case <-ch:
	}

	sort.Slice(children, func(i, j int) bool {
		if children[i].Name == children[j].Name {
			return children[i].Size >= children[j].Size
		}
		return children[i].Name < children[j].Name
	})
	return children
}

type jsonString string

func (s jsonString) String() string {
	return `"` + string(s) + `"`
}

// ContentsFunc is the type of the function that is called to consume the results
// of scanning a single level in the filesystem hierarchy. It should read
// the contents of the supplied channel until that channel is closed.
// Errors, such as failing to access the prefix, are delivered over the channel.
type ContentsFunc func(ctx context.Context, prefix string, info *Info, ch <-chan Contents) ([]Info, error)

// PrefixFunc is the type of the function that is called to determine if a given
// level in the filesystem hiearchy should be further examined or traversed.
// If stop is true then traversal stops at this point, however if a list
// of children is returned, they will be traversed directly rather than
// obtaining the children from the filesystem. This allows for both
// exclusions and incremental processing in conjunction with a database t
// be implemented.
type PrefixFunc func(ctx context.Context, prefix string, info *Info, err error) (stop bool, children []Info, returnErr error)

// Walk traverses the hierarchies specified by each of the roots calling
// prefixFn and contentsFn as it goes. prefixFn will always be called
// before contentsFn for the same prefix, but no other ordering guarantees
// are provided.
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

	// create and prime the concurrency limiter for walking directories.
	walkerLimitCh := make(chan string, w.opts.concurrency*2)
	for i := 0; i < cap(walkerLimitCh); i++ {
		walkerLimitCh <- fmt.Sprintf("%04d", i)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	for _, root := range roots {
		root := root
		walkers.Go(func() error {
			w.walker(ctx, <-walkerLimitCh, root, walkerLimitCh)
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
	return w.errs.Err()
}

func (w *Walker) walkChildren(ctx context.Context, path string, children []Info, limitCh chan string) {
	var wg sync.WaitGroup
	wg.Add(len(children))
	for _, child := range children {
		child := child
		var idx string
		select {
		case idx = <-limitCh:
		case <-ctx.Done():
			return
		default:
			// no concurreny is available fallback to sync.
			p := w.fs.Join(path, child.Name)
			now := time.Now().Format(time.Stamp)
			syncVar.Set(p, jsonString(now))
			w.walker(ctx, now, p, limitCh)
			wg.Done()
			syncVar.Delete(p)
			continue
		}
		go func() {
			w.walker(ctx, idx, w.fs.Join(path, child.Name), limitCh)
			wg.Done()
			limitCh <- idx
		}()
	}
	wg.Wait()

}

func (w *Walker) walker(ctx context.Context, idx string, path string, limitCh chan string) {
	select {
	default:
	case <-ctx.Done():
		return
	}
	walkingVar.Set(idx, jsonString(path))
	defer walkingVar.Delete(idx)
	info, err := w.fs.Stat(ctx, path)
	stop, children, err := w.prefixFn(ctx, path, &info, err)
	w.recordError(path, "stat", err)
	if stop {
		return
	}
	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
		return
	}
	children = w.listLevel(ctx, idx, path, &info)
	if len(children) > 0 {
		w.walkChildren(ctx, path, children, limitCh)
	}
}
