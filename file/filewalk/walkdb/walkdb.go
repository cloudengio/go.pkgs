// Package walkdb provides a simple database for tracking the status
// of a filesystem traversal.
package walkdb

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"cloudeng.io/sync/0.20200414211116-3c1830e6b648/errgroup"
)

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
