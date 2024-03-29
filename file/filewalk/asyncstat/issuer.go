// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package asyncstat

import (
	"context"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/sync/syncsort"
)

// T provides support for issuing asynchronous stat or lstat calls.
type T struct {
	fs      filewalk.FS
	statFn  func(ctx context.Context, filename string) (file.Info, error)
	opts    options
	limitCh chan struct{}
}

// Option is used to configure an asyncstat.T.
type Option func(*options)

type options struct {
	asyncThreshold int
	asyncStats     int
	useStat        bool
	errLogger      ErrorLogger
	latencyTracker LatencyTracker
}

// WithAsyncThreshold sets the threshold at which asynchronous
// stats are used, any directory with less than number of entries
// will be processed synchronously.
// The default is DefaultAsyncThreshold.
func WithAsyncThreshold(threshold int) Option {
	return func(o *options) {
		o.asyncThreshold = threshold
	}
}

// WithAsyncStats sets the total number of asynchronous stats to
// be issued.
// The default is DefaultAsyncStats.
func WithAsyncStats(stats int) Option {
	return func(o *options) {
		if stats > 0 {
			o.asyncStats = stats
		}
	}
}

// WithStat requests that fs.Stat be used instead of fs.LStat.
func WithStat() Option {
	return func(o *options) {
		o.useStat = true
	}
}

// WithLStat requests that fs.LStat be used instead of fs.Stat.
// This is the default.
func WithLStat() Option {
	return func(o *options) {
		o.useStat = false
	}
}

// ErrorLogger is the type of function called when a Stat or Lstat
// return an error.
type ErrorLogger func(ctx context.Context, filename string, err error)

// WithErrorLogger sets the function to be called when an error
// is returned by Stat or Lstat.
func WithErrorLogger(fn ErrorLogger) Option {
	return func(o *options) {
		o.errLogger = fn
	}
}

// LatencyTracker is used to track the latency of Stat or Lstat
// operations.
type LatencyTracker interface {
	Before() time.Time
	After(time.Time)
}

// WithLatencyTracker sets the latency tracker to be used.
func WithLatencyTracker(lt LatencyTracker) Option {
	return func(o *options) {
		o.latencyTracker = lt
	}
}

type nullLatencyTracker struct{}

func (nullLatencyTracker) Before() time.Time { return time.Time{} }
func (nullLatencyTracker) After(time.Time)   {}

var (
	// DefaultAsyncStats is the default maximum number of async stats to be issued
	// when WithAsyncStats is not specified.
	DefaultAsyncStats = 100
	// DefaultAsyncThreshold is the default value for the number of directory
	// entries that must be present before async stats are used when
	// WithAsyncThreshold is not specified.
	DefaultAsyncThreshold = 10
)

// New returns an aysncstat.T that uses the supplied filewalk.FS.
func New(fs filewalk.FS, opts ...Option) *T {
	is := &T{fs: fs}
	is.opts.asyncStats = DefaultAsyncStats
	is.opts.asyncThreshold = DefaultAsyncThreshold
	is.opts.latencyTracker = nullLatencyTracker{}
	is.opts.errLogger = func(context.Context, string, error) {}
	is.statFn = fs.Lstat
	for _, fn := range opts {
		fn(&is.opts)
	}
	if is.opts.useStat {
		is.statFn = fs.Stat
	}
	is.initLimiter()
	return is
}

// Process processes the supplied entries, returning the list of
// children as filewalk.Entry and the list of stat/lstat results as
// a file.InfoList.
func (is *T) Process(ctx context.Context, prefix string, entries []filewalk.Entry) (children, all file.InfoList, err error) {
	if len(entries) < is.opts.asyncThreshold {
		return is.sync(ctx, prefix, entries)
	}
	return is.async(ctx, prefix, entries)
}

func (is *T) callStat(ctx context.Context, filename string) (file.Info, error) {
	start := is.opts.latencyTracker.Before()
	info, err := is.statFn(ctx, filename)
	is.opts.latencyTracker.After(start)
	if err != nil {
		is.opts.errLogger(ctx, filename, err)
	}
	return info, err
}

func (is *T) sync(ctx context.Context, prefix string, entries []filewalk.Entry) (children, all file.InfoList, err error) {
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}
		filename := is.fs.Join(prefix, entry.Name)
		info, err := is.callStat(ctx, filename)
		if err != nil {
			continue
		}
		if entry.IsDir() {
			children = append(children, info)
		}
		all = all.AppendInfo(info)
	}
	return children, all, nil
}

func (is *T) initLimiter() {
	is.limitCh = make(chan struct{}, is.opts.asyncStats)
	for i := 0; i < cap(is.limitCh); i++ {
		is.limitCh <- struct{}{}
	}
}

func (is *T) wait(ctx context.Context) error {
	select {
	case <-is.limitCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (is *T) done() {
	is.limitCh <- struct{}{}
}

type lstatResult struct {
	info file.Info
	err  error
}

func (is *T) async(ctx context.Context, prefix string, entries []filewalk.Entry) (children, all file.InfoList, err error) {
	concurrency := is.opts.asyncStats
	if concurrency > len(entries) {
		concurrency = len(entries)
	}
	g, _ := errgroup.WithContext(ctx)
	g = errgroup.WithConcurrency(g, concurrency)

	// The channel must be large enough to hold all of the items that
	// can be returned.
	ch := make(chan syncsort.Item[lstatResult], len(entries))
	seq := syncsort.NewSequencer(ctx, ch)
	for _, entry := range entries {
		name := entry.Name
		item := seq.NextItem(lstatResult{})
		filename := is.fs.Join(prefix, name)
		if err = is.wait(ctx); err != nil {
			return
		}
		g.Go(func() error {
			info, err := is.callStat(ctx, filename)
			item.V = lstatResult{info, err}
			ch <- item
			is.done()
			return nil
		})
	}
	if err = g.Wait(); err != nil {
		return
	}
	close(ch)
	for seq.Scan() {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}
		res := seq.Item()
		if res.err != nil {
			continue
		}
		if res.info.IsDir() {
			children = append(children, res.info)
		}
		all = all.AppendInfo(res.info)
	}
	err = seq.Err()
	return
}

type Configuration struct {
	AsyncStats     int
	AsyncThreshold int
}

func (is *T) Configuration() Configuration {
	return Configuration{
		AsyncStats:     is.opts.asyncStats,
		AsyncThreshold: is.opts.asyncThreshold,
	}
}
