// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localdb

import (
	"context"
	"fmt"

	"cloudeng.io/file/filewalk"
	"github.com/recoilme/pudge"
)

// Scanner allows for the contents of an instance of Database to be
// enumerated. The database is organized as a key value store that can
// be scanned by range or by prefix.
type Scanner struct {
	pdb     *pudge.Db
	prefix  []byte
	nItems  int // max number of items to read, 0 for all items.
	ifcOpts filewalk.ScannerOptions

	// scan state.
	more            bool
	offset          int
	currentPrefix   string
	currentInfo     filewalk.PrefixInfo
	next            int
	availablePrefix []string
	availableInfo   []filewalk.PrefixInfo
	err             error
}

// ScanOption represents an option used when creating a Scanner.
type ScanOption func(ks *Scanner)

// NewScanner returns a new instance of Scanner.
func NewScanner(db *Database, prefix string, limit int, ifcOpts []filewalk.ScannerOption, opts ...ScanOption) *Scanner {
	sc := &Scanner{
		pdb:    db.prefixdb,
		prefix: []byte(prefix),
		nItems: limit,
	}
	sc.ifcOpts.ScanLimit = 1000
	for _, fn := range ifcOpts {
		fn(&sc.ifcOpts)
	}
	for _, fn := range opts {
		fn(sc)
	}
	if sc.ifcOpts.ScanErrors {
		sc.pdb = db.errordb
	}
	if len(sc.prefix) > 0 {
		pi := &filewalk.PrefixInfo{}
		if err := sc.pdb.Get(sc.prefix, pi); err != nil {
			if err == pudge.ErrKeyNotFound {
				sc.err = fmt.Errorf("start prefix not found, try removing a trailing / and/or make sure it matches a complete prefix or filename")
			} else {
				sc.err = err
			}
		}
	}
	return sc
}

func (sc *Scanner) scanByPrefix(limit int) ([]string, []filewalk.PrefixInfo, error) {
	keys, err := sc.pdb.KeysByPrefix(sc.prefix, limit, sc.offset, !sc.ifcOpts.Descending)
	if err != nil {
		return nil, nil, err
	}
	return sc.processKeys(keys)
}

func (sc *Scanner) scanByRange(limit int) ([]string, []filewalk.PrefixInfo, error) {
	keys, err := sc.pdb.Keys(sc.prefix, limit, sc.offset, !sc.ifcOpts.Descending)
	if err != nil {
		return nil, nil, err
	}
	return sc.processKeys(keys)
}

func (sc *Scanner) processKeys(keys [][]byte) ([]string, []filewalk.PrefixInfo, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}
	sc.offset += len(keys)
	if sc.ifcOpts.KeysOnly {
		return getPrefixes(keys), nil, nil
	}
	return getItems(sc.pdb, keys)
}

func getPrefixes(keys [][]byte) []string {
	prefixes := make([]string, len(keys))
	for i, key := range keys {
		prefixes[i] = string(key)
	}
	return prefixes
}

func getItems(db *pudge.Db, keys [][]byte) ([]string, []filewalk.PrefixInfo, error) {
	prefixes := make([]string, len(keys))
	info := make([]filewalk.PrefixInfo, len(keys))
	for i, key := range keys {
		prefixes[i] = string(key)
		if err := db.Get(key, &info[i]); err != nil {
			return nil, nil, fmt.Errorf("key: %s: err %s", key, err)
		}
	}
	return prefixes, info, nil
}

func (sc *Scanner) fetch() (bool, error) {
	var (
		prefixes []string
		info     []filewalk.PrefixInfo
		err      error
	)
	scanLimit := sc.ifcOpts.ScanLimit
	if sc.nItems > 0 {
		if sc.offset >= sc.nItems {
			return false, nil
		}
		if remaining := sc.nItems - sc.offset; remaining < scanLimit {
			scanLimit = remaining
		}
	}
	if sc.ifcOpts.RangeScan {
		prefixes, info, err = sc.scanByRange(scanLimit)
	} else {
		prefixes, info, err = sc.scanByPrefix(scanLimit)
	}
	if err != nil {
		return false, err
	}
	if len(prefixes) == 0 {
		return false, nil
	}
	sc.availablePrefix = prefixes
	sc.availableInfo = info
	sc.next = 0
	return true, nil
}

// Scan implements filewalk.DatabaseScanner.
func (sc *Scanner) Scan(ctx context.Context) bool {
	if sc.err != nil {
		return false
	}
	select {
	case <-ctx.Done():
		sc.err = ctx.Err()
		return false
	default:
	}
	if !sc.more || sc.next >= len(sc.availablePrefix) {
		more, err := sc.fetch()
		if err != nil || !more {
			sc.err = err
			return false
		}
		sc.more = true
	}
	sc.currentPrefix = sc.availablePrefix[sc.next]
	if !sc.ifcOpts.KeysOnly {
		sc.currentInfo = sc.availableInfo[sc.next]
	}
	sc.next++
	return true
}

var empty = &filewalk.PrefixInfo{}

// PrefixInfo implements filewalk.DatabaseScanner.
func (sc *Scanner) PrefixInfo() (string, *filewalk.PrefixInfo) {
	if sc.ifcOpts.KeysOnly {
		return sc.currentPrefix, empty
	}
	return sc.currentPrefix, &sc.currentInfo
}

// Err rimplements filewalk.DatabaseScanner.
func (sc *Scanner) Err() error {
	return sc.err
}
