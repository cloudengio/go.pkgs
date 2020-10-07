// Package walkdb provides a simple database for tracking the status
// of a filesystem traversal.
package walkdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/os/lockedfile"
	"github.com/recoilme/pudge"
)

const (
	diskUsage        = "__diskUsage"
	numFiles         = "__numFiles"
	numChildren      = "__numChildren"
	totalDiskUsage   = "__totalDiskUsage"
	totalNumFiles    = "__totalNumFiles"
	totalNumChildren = "__totalNumChildren"
	prefixdbFilename = "prefix.pudge"
	statsdbFilename  = "stats.pudge"
	dbLockName       = "walkdb.lock"
	dbLockerInfoName = "walkdb.info"
)

type Database struct {
	dir                string
	prefixdb           *pudge.Db
	prefixdbFilename   string
	statsdb            *pudge.Db
	statsdbFilename    string
	dbLockFilename     string
	dbLockInfoFilename string
	dbMutex            *lockedfile.Mutex
	unlockFn           func()
	diskUsage          sizeHeap
	numFiles           sizeHeap
	numChildren        sizeHeap
	totalDiskUsage     int64
	totalNumChildren   int64
	totalNumFiles      int64
	opts               options
}

var ErrReadonly = errors.New("database is opened in readonly mode")

type Option func(o *Database)

type options struct {
	syncIntervalSeconds int
	readOnly            bool
}

func SyncInterval(interval time.Duration) Option {
	return func(db *Database) {
		i := interval + (500 * time.Millisecond)
		db.opts.syncIntervalSeconds = int(i.Round(time.Second).Seconds())
	}
}

func ReadOnly() Option {
	return func(db *Database) {
		db.opts.readOnly = true
	}
}

type lockFileContents struct {
	User string `json:"user"`
	CWD  string `json:"current_directory"`
	PPID int    `json:"parent_process_pid"`
	PID  int    `json:"process_pid"`
}

func readLockerInfo(filename string) (lockFileContents, error) {
	buf, err := ioutil.ReadFile(filename)
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
		User: os.Getenv("USER"),
		CWD:  cwd,
		PID:  pid,
		PPID: ppid,
	}
	buf, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf, 0666)
}

func newDB(dir string) *Database {
	db := &Database{
		dir:                dir,
		prefixdbFilename:   filepath.Join(dir, prefixdbFilename),
		statsdbFilename:    filepath.Join(dir, statsdbFilename),
		dbLockFilename:     filepath.Join(dir, dbLockName),
		dbLockInfoFilename: filepath.Join(dir, dbLockerInfoName),
		numFiles:           newSizeHeap(),
		numChildren:        newSizeHeap(),
		diskUsage:          newSizeHeap(),
	}
	db.dbMutex = lockedfile.MutexAt(db.dbLockFilename)
	return db
}

func (db *Database) writeLock() error {
	unlock, err := db.dbMutex.Lock()
	if err == nil {
		db.unlockFn = unlock
		return writeLockerInfo(db.dbLockInfoFilename)
	}
	info, nerr := readLockerInfo(db.dbLockInfoFilename)
	owner := ""
	if nerr == nil {
		if str, err := json.MarshalIndent(info, "\t", "  "); err != nil {
			owner = "\n\tcurrent lock info\n" + string(str) + "\n"
		}
	}
	return fmt.Errorf("failed to lock %v: %v%v", db.dir, err, owner)
}

func (db *Database) readLock() error {
	unlock, err := db.dbMutex.RLock()
	if err != nil {
		return err
	}
	db.unlockFn = unlock
	return nil
}

func (db *Database) unlock() error {
	err := os.Remove(db.dbLockInfoFilename)
	db.unlockFn()
	return err
}

func Open(dir string, opts ...Option) (*Database, error) {
	db := newDB(dir)
	db.opts.syncIntervalSeconds = 30
	for _, fn := range opts {
		fn(db)
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
	pdb, err := pudge.Open(db.prefixdbFilename, &cfg)
	if err != nil {
		return nil, err
	}
	sdb, err := pudge.Open(db.statsdbFilename, &cfg)
	if err != nil {
		return nil, err
	}
	db.prefixdb = pdb
	db.statsdb = sdb
	if err := db.loadStats(); err != nil {
		pdb.Close()
		sdb.Close()
		return nil, fmt.Errorf("failed to load stats: %v", err)
	}
	return db, nil
}

func (db *Database) loadStats() error {
	for _, stat := range []struct {
		key  string
		stat interface{}
	}{
		{numFiles, &db.numFiles},
		{numChildren, &db.numChildren},
		{diskUsage, &db.diskUsage},
		{totalNumFiles, &db.totalNumFiles},
		{totalNumChildren, &db.totalNumChildren},
		{totalDiskUsage, &db.totalDiskUsage},
	} {
		ok, err := db.statsdb.Has(stat.key)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := db.statsdb.Get(stat.key, stat.stat); err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) updateStats(prefix string, info PrefixInfo) {
	deltaFiles := db.numFiles.update(prefix, int64(len(info.Files)))
	deltaChildren := db.numChildren.update(prefix, int64(len(info.Children)))
	deltaUsage := db.diskUsage.update(prefix, info.DiskUsage)
	db.totalNumFiles += deltaFiles
	db.totalNumChildren += deltaChildren
	db.totalDiskUsage += deltaUsage
}

func (db *Database) saveStats() error {
	if db.opts.readOnly {
		return ErrReadonly
	}
	errs := errors.M{}
	db.diskUsage.init()
	db.numFiles.init()
	db.numChildren.init()
	errs.Append(db.statsdb.Set(diskUsage, &db.diskUsage))
	errs.Append(db.statsdb.Set(numFiles, &db.numFiles))
	errs.Append(db.statsdb.Set(numChildren, &db.numChildren))
	errs.Append(db.statsdb.Set(totalDiskUsage, &db.totalDiskUsage))
	errs.Append(db.statsdb.Set(totalNumFiles, &db.totalNumFiles))
	errs.Append(db.statsdb.Set(totalNumChildren, &db.totalNumChildren))
	return errs.Err()
}

func (db *Database) Persist() error {
	if db.opts.readOnly {
		return ErrReadonly
	}
	errs := errors.M{}
	errs.Append(db.saveStats())
	errs.Append(db.statsdb.Close())
	errs.Append(db.prefixdb.Close())
	return errs.Err()
}

func (db *Database) Set(prefix string, info PrefixInfo) error {
	if db.opts.readOnly {
		return ErrReadonly
	}
	db.updateStats(prefix, info)
	return db.prefixdb.Set(prefix, &info)
}

func (db *Database) Get(prefix string) (*PrefixInfo, bool, error) {
	info := new(PrefixInfo)
	if err := db.prefixdb.Get(prefix, info); err != nil {
		if err == pudge.ErrKeyNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return info, true, nil
}

// UnchangedDirInfo returns true if the newly obtainined filewalk.Info is
// unchanged from that in the database.
func (db *Database) UnchangedDirInfo(prefix string, info *filewalk.Info) (*PrefixInfo, bool, error) {
	pi, ok, err := db.Get(prefix)
	if err != nil || !ok {
		return nil, false, err
	}
	unchanged := pi.ModTime == info.ModTime &&
		filewalk.FileMode(pi.Mode) == info.Mode
	if unchanged {
		return pi, true, nil
	}
	return nil, false, nil
}

type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
}

type PrefixInfo struct {
	ModTime   time.Time
	Mode      uint32
	Size      int64
	Children  []*filewalk.Info
	Files     []FileInfo
	DiskUsage int64
	Err       string
}

func topN(m []Metric, n int) []Metric {
	if len(m) <= n {
		return m
	}
	return m[:n]
}

func (db *Database) FileCounts(n int) []Metric {
	return db.numFiles.TopN(n)
}

func (db *Database) ChildCounts(n int) []Metric {
	return db.numChildren.TopN(n)
}

func (db *Database) DiskUsage(n int) []Metric {
	return db.diskUsage.TopN(n)
}

func (db *Database) Totals() (files, children, diskUsage int64) {
	files, children, diskUsage = db.totalNumFiles, db.totalNumChildren, db.totalDiskUsage
	return
}
