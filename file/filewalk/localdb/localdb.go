// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package localdb provides an implementation of filewalk.Database that
// uses a local key/value store currently based on github.com/recoilme/pudge.
package localdb

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/os/lockedfile"
	"cloudeng.io/os/userid"
	"cloudeng.io/sync/errgroup"
	"github.com/cosnicolaou/pudge"
)

var dbStatus = expvar.NewMap("cloudeng.io/file/filewalk.pudgedb")

const (
	globalStatsKey   = "__globalStats"
	usersListKey     = "__userList"
	groupsListKey    = "__groupList"
	prefixdbFilename = "prefix.pudge"
	statsdbFilename  = "stats.pudge"
	userdbFilename   = "users.pudge"
	groupdbFilename  = "groups.pudge"
	errordbFilename  = "errors.pudge"
	dbLockName       = "db.lock"
	dbLockerInfoName = "db.info"
)

// Database represents an on-disk database that stores information
// and statistics for filesystem directories/prefixes. The database
// supports read-write and read-only modes of access.
type Database struct {
	opts               options
	dir                string
	prefixdb           *pudge.Db
	statsdb            *pudge.Db
	errordb            *pudge.Db
	userdb             *pudge.Db
	groupdb            *pudge.Db
	dbLockFilename     string
	dbLockInfoFilename string
	dbMutex            *lockedfile.Mutex
	unlockedMu         sync.Mutex
	unlocked           bool   // GUARDED_BY(unlockedMu)
	unlockFn           func() // GUARDED_BY(unlockedMu)
	globalStats        *statsCollection
	userStats          *perItemStats
	groupStats         *perItemStats
}

// ErrReadonly is returned if an attempt is made to write to a database
// opened in read-only mode.
var ErrReadonly = errors.New("database is opened in readonly mode")

// DatabaseOption represents a specific option accepted by Open.
type DatabaseOption func(o *Database)

type options struct {
	readOnly            bool
	errorsOnly          bool
	resetStats          bool
	syncIntervalSeconds int
	lockRetryDelay      time.Duration
	tryLock             bool
}

// SyncInterval set the interval at which the database is to be
// persisted to disk.
func SyncInterval(interval time.Duration) DatabaseOption {
	return func(db *Database) {
		if interval == 0 {
			db.opts.syncIntervalSeconds = 60
			return
		}
		i := interval + (500 * time.Millisecond)
		db.opts.syncIntervalSeconds = int(i.Round(time.Second).Seconds())
	}
}

// TryLock returns an error if the database cannot be locked within
// the delay period.
func TryLock() DatabaseOption {
	return func(db *Database) {
		db.opts.tryLock = true
	}
}

// LockStatusDelay sets the delay between checking the status of acquiring a
// lock on the database.
func LockStatusDelay(d time.Duration) DatabaseOption {
	return func(db *Database) {
		db.opts.lockRetryDelay = d
	}
}

type lockFileContents struct {
	User string `json:"user"`
	CWD  string `json:"current_directory"`
	PPID int    `json:"parent_process_pid"`
	PID  int    `json:"process_pid"`
}

func readLockerInfo(filename string) (lockFileContents, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return lockFileContents{}, err
	}
	var contents lockFileContents
	err = json.Unmarshal(buf, &contents)
	return contents, err
}

func writeLockerInfo(filename string) error {
	cwd, _ := os.Getwd()
	pid := os.Getpid()
	ppid := os.Getppid()
	contents := lockFileContents{
		User: userid.GetCurrentUser(),
		CWD:  cwd,
		PID:  pid,
		PPID: ppid,
	}
	buf, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, buf, 0666)
}

func newDB(dir string) *Database {
	db := &Database{
		dir:                dir,
		dbLockFilename:     filepath.Join(dir, dbLockName),
		dbLockInfoFilename: filepath.Join(dir, dbLockerInfoName),
		globalStats:        newStatsCollection(globalStatsKey),
		userStats:          newPerItemStats(usersListKey),
		groupStats:         newPerItemStats(groupsListKey),
	}
	db.opts.lockRetryDelay = time.Second * 5
	if err := os.MkdirAll(dir, 0770); err != nil {
		panic(err)
	}
	db.dbMutex = lockedfile.MutexAt(db.dbLockFilename)
	return db
}

func lockerInfo(filename string) string {
	info, err := readLockerInfo(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return err.Error()
		}
		return fmt.Sprintf("failed to obtain locker info from %v: %v", filename, err)
	}
	str, _ := json.MarshalIndent(info, "", "  ")
	return string(str)
}

func lockerErrorInfo(dir, filename string, err error) error {
	return fmt.Errorf("failed to lock %v: %v\nlock info from: %v:\n%v", dir, err, filename, lockerInfo(filename))
}

func (db *Database) acquireLock(ctx context.Context, readOnly bool, tryDelay time.Duration, tryLock bool) error {
	type lockResult struct {
		unlock func()
		err    error
	}
	lockType := "write "
	if readOnly {
		lockType = "read "
	}
	ch := make(chan lockResult)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		var unlock func()
		var err error
		if readOnly {
			unlock, err = db.dbMutex.RLock()
		} else {
			unlock, err = db.dbMutex.Lock()
		}
		select {
		case ch <- lockResult{unlock, err}:
		case <-ctx.Done():
			if unlock != nil {
				db.setUnlockFn(unlock)
				db.unlock() //nolint:errcheck
			}
		}
	}()
	for {
		select {
		case lr := <-ch:
			if lr.err == nil {
				db.setUnlockFn(func() { lr.unlock(); cancel() })
				if readOnly {
					return nil
				}
				return writeLockerInfo(db.dbLockInfoFilename)
			}
			cancel()
			return lockerErrorInfo(db.dir, db.dbLockInfoFilename, lr.err)
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case <-time.After(tryDelay):
			if tryLock {
				err := fmt.Errorf("failed to acquire %slock after %v", lockType, tryDelay)
				cancel()
				return lockerErrorInfo(db.dir, db.dbLockInfoFilename, err)
			}
			fmt.Fprintf(os.Stderr, "waiting to acquire %slock: lock info from %s:\n", lockType, db.dbLockInfoFilename)
			fmt.Fprintf(os.Stderr, "%s\n", lockerInfo(db.dbLockInfoFilename))
			tryDelay *= 2
			if tryDelay > time.Minute*10 {
				tryDelay = time.Minute * 10
			}
		}
	}
}

func (db *Database) setUnlockFn(fn func()) {
	db.unlockedMu.Lock()
	defer db.unlockedMu.Unlock()
	db.unlockFn = fn
}

func (db *Database) unlock() error {
	db.unlockedMu.Lock()
	defer db.unlockedMu.Unlock()
	if db.unlocked {
		return nil
	}
	err := os.Remove(db.dbLockInfoFilename)
	db.unlockFn()
	db.unlocked = true
	return err
}

type jsonString string

func (s jsonString) String() string {
	return `"` + string(s) + `"`
}

func Open(ctx context.Context, dir string, ifcOpts []filewalk.DatabaseOption, opts ...DatabaseOption) (filewalk.Database, error) {
	db := newDB(dir)
	var dbOpts filewalk.DatabaseOptions
	for _, fn := range ifcOpts {
		fn(&dbOpts)
	}
	db.opts.readOnly = dbOpts.ReadOnly
	db.opts.errorsOnly = dbOpts.ErrorsOnly
	db.opts.resetStats = dbOpts.ResetStats
	db.opts.lockRetryDelay = time.Minute
	for _, fn := range opts {
		fn(db)
	}

	if err := db.acquireLock(ctx, db.opts.readOnly, db.opts.lockRetryDelay, db.opts.tryLock); err != nil {
		return nil, err
	}

	cfg := pudge.Config{
		StoreMode:    0,
		FileMode:     0666,
		DirMode:      0777,
		SyncInterval: db.opts.syncIntervalSeconds,
	}
	if db.opts.readOnly {
		cfg.SyncInterval = 0
	}

	type dbPair struct {
		dbp  **pudge.Db
		name string
	}

	var toOpen = []dbPair{
		{&db.errordb, errordbFilename},
	}
	if !db.opts.errorsOnly {
		toOpen = append(toOpen,
			dbPair{&db.prefixdb, prefixdbFilename},
			dbPair{&db.statsdb, statsdbFilename},
			dbPair{&db.userdb, userdbFilename},
			dbPair{&db.groupdb, groupdbFilename},
		)
	}
	var gdb errgroup.T
	for _, dbg := range toOpen {
		name := filepath.Join(dir, dbg.name)
		dbp := dbg.dbp
		gdb.Go(func() error {
			var err error
			dbStatus.Set(name, jsonString("opening"))
			*dbp, err = pudge.Open(name, &cfg)
			if err != nil {
				dbStatus.Set(name, jsonString("opened: failed: "+err.Error()))
				return err
			}
			dbStatus.Set(name, jsonString("opened"))
			return nil
		})
	}
	if err := gdb.Wait(); err != nil {
		db.closeAll() //nolint:errcheck
		return nil, err
	}
	if db.opts.errorsOnly {
		return db, nil
	}
	if db.opts.resetStats {
		return db, nil
	}
	var gstats errgroup.T
	gstats.Go(func() error {
		if err := db.globalStats.loadOrInit(db.statsdb, globalStatsKey); err != nil {
			return fmt.Errorf("failed to load stats: %v", err)
		}
		return nil
	})
	gstats.Go(func() error {
		if err := db.userStats.loadItemList(db.userdb); err != nil {
			return fmt.Errorf("failed to load user list: %v", err)
		}
		return nil
	})
	gstats.Go(func() error {
		if err := db.groupStats.loadItemList(db.groupdb); err != nil {
			return fmt.Errorf("failed to load group list: %v", err)
		}
		return nil
	})
	if err := gstats.Wait(); err != nil {
		db.closeAll() //nolint:errcheck
		return nil, err
	}
	return db, nil
}

func (db *Database) closeAll() error {
	errs := errors.M{}
	closer := func(db *pudge.Db) {
		if db != nil {
			errs.Append(db.Close())
		}
	}
	closer(db.prefixdb)
	closer(db.statsdb)
	closer(db.errordb)
	closer(db.userdb)
	closer(db.groupdb)
	db.unlock() //nolint:errcheck
	return errs.Err()
}

func (db *Database) saveStats() error { //nolint:unused
	if db.opts.readOnly {
		return ErrReadonly
	}
	return db.globalStats.save(db.statsdb, globalStatsKey)
}

func (db *Database) CompactAndClose(_ context.Context) error {
	if db.opts.readOnly {
		return fmt.Errorf("database is readonly")
	}
	g := errgroup.T{}
	g.Go(func() error {
		return db.prefixdb.CompactAndClose()
	})
	g.Go(func() error {
		return db.statsdb.CompactAndClose()
	})
	g.Go(func() error {
		return db.userdb.CompactAndClose()
	})
	g.Go(func() error {
		return db.groupdb.CompactAndClose()
	})
	g.Go(func() error {
		return db.errordb.CompactAndClose()
	})
	err := g.Wait()
	db.unlock() //nolint:errcheck
	return err
}

func (db *Database) Close(ctx context.Context) error {
	if !db.opts.readOnly {
		return db.Save(ctx)
	}
	return db.closeAll()
}

func (db *Database) Save(_ context.Context) error {
	if db.opts.readOnly {
		return ErrReadonly
	}
	g := errgroup.T{}
	g.Go(func() error {
		return db.globalStats.save(db.statsdb, globalStatsKey)
	})
	g.Go(func() error {
		return db.userStats.save(db.userdb)
	})
	g.Go(func() error {
		return db.groupStats.save(db.groupdb)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return db.closeAll()
}

func (db *Database) Set(_ context.Context, prefix string, info *filewalk.PrefixInfo) error {
	if db.opts.readOnly {
		return ErrReadonly
	}
	db.globalStats.update(prefix, info)
	errs := errors.M{}
	errs.Append(db.prefixdb.Set(prefix, info))
	errs.Append(db.userStats.updateStats(db.userdb, prefix, info.UserID, info))
	errs.Append(db.groupStats.updateStats(db.groupdb, prefix, info.GroupID, info))
	err := errs.Err()
	switch {
	case err == nil && len(info.Err) == 0:
		return nil
	case err == nil && len(info.Err) != 0:
		errs.Append(db.errordb.Set(prefix, info))
		return errs.Err()
	}
	ninfo := &filewalk.PrefixInfo{
		ModTime: time.Now(),
	}
	timestamp := time.Now().Format(time.StampMilli)
	if len(info.Err) != 0 {
		ninfo.Err = fmt.Sprintf("%v: %v: failed to write to database: %v", timestamp, info.Err, err)
	} else {
		ninfo.Err = fmt.Sprintf("%v: failed to write to database: %v", timestamp, err)
	}
	errs.Append(db.errordb.Set(prefix, ninfo))
	return errs.Err()
}

func (db *Database) Get(_ context.Context, prefix string, info *filewalk.PrefixInfo) (bool, error) {
	if err := db.prefixdb.Get(prefix, info); err != nil {
		if err == pudge.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (db *Database) Delete(ctx context.Context, separator string, prefixes []string, recurse bool) (int, error) {
	errs := &errors.M{}
	deleted := 0
	for _, prefix := range prefixes {
		deletions := db.delete(ctx, separator, prefix, recurse, errs)
		nd, err := db.prefixdb.DeleteKeys(deletions)
		if err != nil {
			errs.Append(fmt.Errorf("%v: %v", deletions[nd+1], err))
			continue
		}
		deleted++
	}
	return deleted, errs.Err()
}

func (db *Database) DeleteErrors(_ context.Context, prefixes []string) (int, error) {
	deletions := make([]interface{}, 0, len(prefixes))
	for _, prefix := range prefixes {
		deletions = append(deletions, prefix)
	}
	return db.errordb.DeleteKeys(deletions)
}

func (db *Database) delete(ctx context.Context, separator, prefix string, recurse bool, errs *errors.M) []interface{} {
	var existing filewalk.PrefixInfo
	err := db.prefixdb.Get(prefix, &existing)
	if err != nil {
		errs.Append(fmt.Errorf("get: %v: %v", prefix, err))
		return nil
	}
	var deletions []interface{}
	if recurse {
		deletions = make([]interface{}, 0, len(existing.Children))
		for _, child := range existing.Children {
			deletions = append(deletions, db.delete(ctx, separator, prefix+separator+child.Name(), true, errs)...)
		}
	}
	db.globalStats.remove(prefix)
	errs.Append(db.userStats.remove(db.userdb, prefix, existing.UserID))
	errs.Append(db.groupStats.remove(db.groupdb, prefix, existing.GroupID))
	return append(deletions, prefix)
}

func (db *Database) NewScanner(prefix string, limit int, opts ...filewalk.ScannerOption) filewalk.DatabaseScanner {
	return NewScanner(db, prefix, limit, opts)
}

func (db *Database) UserIDs(_ context.Context) ([]string, error) {
	return db.userStats.itemKeys, nil
}

func (db *Database) GroupIDs(_ context.Context) ([]string, error) {
	return db.groupStats.itemKeys, nil
}

func getMetricNames() []filewalk.MetricName {
	metrics := []filewalk.MetricName{
		filewalk.TotalFileCount,
		filewalk.TotalPrefixCount,
		filewalk.TotalDiskUsage,
		filewalk.TotalErrorCount,
	}
	sort.Slice(metrics, func(i, j int) bool {
		return string(metrics[i]) < string(metrics[j])
	})
	return metrics
}

func (db *Database) statsForDb(pdb *pudge.Db, filename, name, desc string) (filewalk.DatabaseStats, error) {
	count, err := pdb.Count()
	if err != nil {
		return filewalk.DatabaseStats{}, fmt.Errorf("failed to get # entries for %v: %v", name, err)
	}
	size, err := pdb.FileSize()
	if err != nil {
		return filewalk.DatabaseStats{}, fmt.Errorf("failed to size %v for %v: %v", filename, name, err)
	}
	return filewalk.DatabaseStats{
		Name:        name,
		Description: desc,
		NumEntries:  int64(count),
		Size:        size,
	}, nil
}

func (db *Database) Stats() ([]filewalk.DatabaseStats, error) {
	stats := []filewalk.DatabaseStats{}
	errs := errors.M{}
	for _, dbi := range []struct {
		pdb                  *pudge.Db
		filename, name, desc string
	}{
		{db.prefixdb, prefixdbFilename, "prefixes", "database containing information for every prefix"},
		{db.statsdb, statsdbFilename, "stats", "database containing statistics for every prefix"},
		{db.userdb, userdbFilename, "user-stats", "database containing statistics for every prefix partitioned by user"},
		{db.groupdb, groupdbFilename, "group-stats", "database containing statistics for every prefix partioned by group"},
		{db.errordb, errordbFilename, "errors", "database containing information on errors encountered to date"},
	} {
		stat, err := db.statsForDb(dbi.pdb, dbi.filename, dbi.name, dbi.desc)
		stats = append(stats, stat)
		errs.Append(err)
	}
	return stats, errs.Err()
}

func (db *Database) Metrics() []filewalk.MetricName {
	return getMetricNames()
}

func metricOptions(opts []filewalk.MetricOption) filewalk.MetricOptions {
	var o filewalk.MetricOptions
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

func (db *Database) statsCollectionForOption(o filewalk.MetricOptions) (*statsCollection, error) {
	if o.Global {
		return db.globalStats, nil
	}
	switch {
	case len(o.UserID) > 0:
		return db.userStats.statsForItem(db.userdb, o.UserID)
	case len(o.GroupID) > 0:
		return db.userStats.statsForItem(db.groupdb, o.GroupID)
	}
	return nil, fmt.Errorf("unrecognised options %#v", o)
}

func (db *Database) Total(_ context.Context, name filewalk.MetricName, opts ...filewalk.MetricOption) (int64, error) {
	o := metricOptions(opts)
	sc, err := db.statsCollectionForOption(o)
	if err != nil {
		return -1, err
	}
	return sc.total(name)
}

func (sc *statsCollection) total(name filewalk.MetricName) (int64, error) {
	switch name {
	case filewalk.TotalFileCount:
		return sc.NumFiles.Sum(), nil
	case filewalk.TotalPrefixCount:
		return sc.NumChildren.Sum(), nil
	case filewalk.TotalDiskUsage:
		return sc.DiskUsage.Sum(), nil
	case filewalk.TotalErrorCount:
		return sc.NumErrors, nil
	}
	return -1, fmt.Errorf("unsupported metric: %v", name)
}

func (db *Database) TopN(_ context.Context, name filewalk.MetricName, n int, opts ...filewalk.MetricOption) ([]filewalk.Metric, error) {
	o := metricOptions(opts)
	sc, err := db.statsCollectionForOption(o)
	if err != nil {
		return nil, err
	}
	return sc.topN(name, n)
}

func topNMetrics(top []struct {
	K string
	V int64
}) []filewalk.Metric {
	m := make([]filewalk.Metric, len(top))
	for i, kv := range top {
		m[i] = filewalk.Metric{Prefix: kv.K, Value: kv.V}
	}
	return m
}

func (sc *statsCollection) topN(name filewalk.MetricName, n int) ([]filewalk.Metric, error) {
	switch name {
	case filewalk.TotalFileCount:
		return topNMetrics(sc.NumFiles.TopN(n)), nil
	case filewalk.TotalPrefixCount:
		return topNMetrics(sc.NumChildren.TopN(n)), nil
	case filewalk.TotalDiskUsage:
		return topNMetrics(sc.DiskUsage.TopN(n)), nil
	case filewalk.TotalErrorCount:
		return nil, nil
	}
	return nil, fmt.Errorf("unsupported metric: %v", name)
}
