// Package walkdb provides a simple database for tracking the status
// of a filesystem traversal.
package walkdb

import (
	"container/heap"
	"fmt"
	"time"

	"cloudeng.io/errors"
	"github.com/recoilme/pudge"
)

const (
	prefixSize               = "__prefixSize"
	cumulativePrefixSize     = "__cumulativePrefixSize"
	prefixFiles              = "__prefixFiles"
	cumulativePrefixChildren = "__cumulativeFiles"
)

type Database struct {
	pdb             *pudge.Db
	prefixSize      sizeHeap
	cumulativeSize  sizeHeap
	prefixFiles     sizeHeap
	cumulativeFiles sizeHeap
	opts            options
}

type Option func(o *options)

type options struct {
	syncIntervalSeconds int
}

func SyncInterval(interval time.Duration) Option {
	return func(o *options) {
		i := interval + (500 * time.Millisecond)
		o.syncIntervalSeconds = int(i.Round(time.Second).Seconds())
	}
}

func Open(dbfile string, opts ...Option) (*Database, error) {
	db := &Database{}
	db.opts.syncIntervalSeconds = 30
	for _, fn := range opts {
		fn(&db.opts)
	}
	cfg := pudge.Config{
		StoreMode:    2, // memory first, then file.
		FileMode:     0666,
		DirMode:      0777,
		SyncInterval: db.opts.syncIntervalSeconds,
	}
	pdb, err := pudge.Open(dbfile, &cfg)
	if err != nil {
		return nil, err
	}
	db.pdb = pdb
	if err := db.loadStats(); err != nil {
		pdb.Close()
		return nil, fmt.Errorf("failed to load stats: %v", err)
	}
	return db, nil
}

func (db *Database) loadStats() error {
	for _, stat := range []struct {
		key  string
		stat *sizeHeap
	}{
		{prefixSize, &db.prefixSize},
		{cumulativePrefixSize, &db.cumulativeSize},
		{prefixFiles, &db.prefixFiles},
		{cumulativePrefixChildren, &db.cumulativeFiles},
	} {
		ok, err := db.pdb.Has(stat.key)
		if err != nil {
			return err
		}
		if !ok {
			heap.Init(stat.stat)
			continue
		}
		if err := db.pdb.Get(stat.key, stat.stat); err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) saveStats() error {
	errs := errors.M{}
	errs.Append(db.pdb.Set(prefixSize, &db.prefixSize))
	errs.Append(db.pdb.Set(cumulativePrefixSize, &db.cumulativeSize))
	errs.Append(db.pdb.Set(prefixFiles, &db.prefixFiles))
	errs.Append(db.pdb.Set(cumulativePrefixChildren, &db.cumulativeFiles))
	return errs.Err()
}

func (db *Database) Persist() error {
	errs := errors.M{}
	errs.Append(db.saveStats())
	errs.Append(db.pdb.Close())
	return errs.Err()
}

func (db *Database) updateStats(prefix string, info PrefixInfo) {
	db.prefixSize.update(prefix, info.Size)
	db.cumulativeSize.update(prefix, info.CumulativeSize)
	db.prefixFiles.update(prefix, len(info.Files))
	db.cumulativeFiles.update(prefix, info.CumulativeFiles)
}

func (db *Database) Set(prefix string, info PrefixInfo) error {
	db.updateStats(prefix, info)
	return db.pdb.Set(prefix, &info)
}

func (db *Database) Get(prefix string) (*PrefixInfo, error) {
	info := new(PrefixInfo)
	if err := db.pdb.Get(prefix, info); err != nil {
		return nil, err
	}
	return info, nil
}

type FileInfo struct {
	Name    string
	Size    int
	ModTime time.Time
}

type PrefixInfo struct {
	Children        []string
	Files           []FileInfo
	CumulativeFiles int
	Size            int
	CumulativeSize  int
}

/*
type fileContents struct {
	Records map[string]json.RawMessage `json:"r,omitempty"`
}

type Database struct {
	mu    sync.Mutex
	root  string
	files []fileContents
}

func newDB(dir string) *Database {
	return &Database{
		root:  dir,
		files: make([]fileContents, 16*16), // 2 char prefix from sha1 of dirname.
	}
}

func New(dir string) (*Database, error) {
	fi, err := os.Stat(dir)
	if err == nil {
		if !fi.IsDir() {
			return nil, fmt.Errorf("%v is not a directory", dir)
		}
		return newDB(dir), nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return newDB(dir), nil
}

func (db *Database) useOrLoad(prefix string) (int, error) {
	file, idx := db.fileFor(prefix)
	records := db.files[idx].Records
	if records == nil {
		rd, err := os.Open(file)
		if err != nil {
			return -1, err
		}
		defer rd.Close()
		records = map[string]json.RawMessage{}
		dec := json.NewDecoder(rd)
		if err := dec.Decode(&records); err != nil {
			return -1, fmt.Errorf("%v: failed to json decode contents: %v", file, err)
		}
		db.files[idx].Records = records
	}
	return idx, nil
}

func (db *Database) Lookup(ctx context.Context, prefix string) (json.RawMessage, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	idx, err := db.useOrLoad(prefix)
	if err != nil {
		return nil, err
	}
	return db.files[idx].Records[prefix], nil
}

func (db *Database) Update(ctx context.Context, prefix string, data json.RawMessage) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	idx, err := db.useOrLoad(prefix)
	if err != nil {
		return err
	}
	db.files[idx].Records[prefix] = data
	return nil
}

func (db *Database) Persist(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	writers, ctx := errgroup.WithContext(ctx)
	writers = errgroup.WithConcurrency(writers, 4)
	for i := range db.files {
		writers.Go(func() error {
			return db.writeContents(i)
		})
	}
	return writers.Err()
}

func (db *Database) writeContents(idx int) error {
	filename := filepath.Join(db.root, fmt.Sprintf("02x", idx))
	wr, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer wr.Close()
	enc := json.NewEncoder(wr)
	return enc.Encode(db.files[idx])
}

func (db *Database) fileFor(path string) (string, int) {
	a := hex.EncodeToString(hashSHA1(path)[:2])
	idx, _ := strconv.ParseInt(a, 16, 64)
	return filepath.Join(db.root, a), int(idx)
}

func hashSHA1(path string) []byte {
	h := sha1.New()
	h.Write([]byte(path))
	return h.Sum(nil)
}
*/
