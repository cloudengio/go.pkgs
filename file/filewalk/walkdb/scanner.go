package walkdb

import (
	"fmt"

	"github.com/recoilme/pudge"
)

type Scanner struct {
	db              *Database
	prefix          []byte
	ascending       bool
	more            bool
	limit, offset   int
	availablePrefix []string
	availableInfo   []PrefixInfo
	current         int
	err             error
}

func (db *Database) NewScanner(prefix string, ascending bool) *Scanner {
	sc := &Scanner{
		prefix:    []byte(prefix),
		db:        db,
		limit:     1000,
		ascending: ascending,
	}
	sc.fetch()
	return sc
}

func scanByPrefix(db *pudge.Db, prefix []byte, limit, offset int, ascending bool) ([]string, []PrefixInfo, error) {
	keys, err := db.KeysByPrefix(prefix, limit, offset, ascending)
	if err != nil {
		return nil, nil, err
	}
	prefixes := make([]string, len(keys))
	info := make([]PrefixInfo, len(keys))
	for i, key := range keys {
		prefixes[i] = string(key)
		if err := db.Get(key, &info[i]); err != nil {
			return nil, nil, fmt.Errorf("key: %s: err %s", key, err)
		}
	}
	return prefixes, info, nil
}

func (sc *Scanner) fetch() {
	prefixes, info, err := scanByPrefix(sc.db.prefixdb, sc.prefix, sc.limit, sc.offset, sc.ascending)
	if err != nil {
		sc.err = err
		return
	}
	if len(prefixes) == 0 {
		sc.more = false
		return
	}
	sc.offset += len(prefixes)
	sc.availablePrefix = prefixes
	sc.availableInfo = info
	sc.current = 0
	sc.more = true
}

func (sc *Scanner) Scan() bool {
	if !sc.more || sc.err != nil {
		return false
	}
	sc.current++
	if sc.current < len(sc.availablePrefix) {
		return true
	}
	sc.fetch()
	return sc.more && sc.err == nil
}

func (sc *Scanner) Item() (string, PrefixInfo) {
	return sc.availablePrefix[sc.current], sc.availableInfo[sc.current]
}

func (sc *Scanner) Err() error {
	return sc.err
}
