// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localdb

import (
	"fmt"
	"sort"

	"cloudeng.io/algo/container/heap"
	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
	"github.com/recoilme/pudge"
)

type statsCollection struct {
	StatsKey    string
	DiskUsage   *heap.KeyedInt64
	NumFiles    *heap.KeyedInt64
	NumChildren *heap.KeyedInt64
	NumErrors   int64
}

func newStatsCollection(key string) *statsCollection {
	return &statsCollection{
		StatsKey:    key,
		NumFiles:    heap.NewKeyedInt64(heap.Descending),
		NumChildren: heap.NewKeyedInt64(heap.Descending),
		DiskUsage:   heap.NewKeyedInt64(heap.Descending),
	}
}

func (sc *statsCollection) loadOrInit(db *pudge.Db, key string) error {
	for _, kv := range []struct {
		key string
		val interface{}
	}{
		{key + ".key", &sc.StatsKey},
		{key + ".files", &sc.NumFiles},
		{key + ".children", &sc.NumChildren},
		{key + ".usage", &sc.DiskUsage},
	} {
		if err := db.Get(kv.key, kv.val); err != nil {
			if err != pudge.ErrKeyNotFound {
				return err
			}
		}
	}
	return nil
}

func (sc *statsCollection) save(db *pudge.Db, key string) error {
	errs := errors.M{}
	errs.Append(db.Set(key+".key", sc.StatsKey))
	errs.Append(db.Set(key+".files", sc.NumFiles))
	errs.Append(db.Set(key+".children", sc.NumChildren))
	errs.Append(db.Set(key+".usage", sc.DiskUsage))
	return errs.Err()
}

func (sc *statsCollection) update(prefix string, info *filewalk.PrefixInfo) {
	if info.DiskUsage > 0 {
		sc.DiskUsage.Update(prefix, info.DiskUsage)
	}
	if len(info.Files) > 0 {
		sc.NumFiles.Update(prefix, int64(len(info.Files)))
	}
	if len(info.Children) > 0 {
		sc.NumChildren.Update(prefix, int64(len(info.Children)))
	}
	if len(info.Err) != 0 {
		sc.NumErrors++
	}
}

func (sc *statsCollection) remove(prefix string) {
	sc.DiskUsage.Remove(prefix)
	sc.NumChildren.Remove(prefix)
	sc.NumFiles.Remove(prefix)
}

// perItemStats provides granular, keyed, stats for providing
// per user, or per group stats. It maintains a list of items
// as well as per-item data so that per-item data can be loaded
// incrementally as needed rather than all at once at startup.
type perItemStats struct {
	itemListKey string
	stats       map[string]*statsCollection
	itemKeys    []string
}

func newPerItemStats(name string) *perItemStats {
	return &perItemStats{
		stats: make(map[string]*statsCollection),
	}
}

func dumpKeys(db *pudge.Db) {
	keys, err := db.Keys(nil, 0, 0, true)
	fmt.Printf("ERR: %v %v\n", err, len(keys))
	for _, k := range keys {
		fmt.Printf("%s\n", k)
	}
}

func (pu *perItemStats) loadItemList(db *pudge.Db) error {
	if err := db.Get(pu.itemListKey, &pu.itemKeys); err != nil && err != pudge.ErrKeyNotFound {
		return err
	}
	return nil
}

func (pu *perItemStats) initStatsForItem(db *pudge.Db, item string) (*statsCollection, error) {
	sdb, ok := pu.stats[item]
	if !ok {
		sc := newStatsCollection(item)
		err := sc.loadOrInit(db, item)
		if err != nil {
			return nil, err
		}
		pu.stats[item] = sc
		pu.itemKeys = append(pu.itemKeys, item)
		return pu.stats[item], err
	}
	return sdb, nil
}

func (pu *perItemStats) statsForItem(db *pudge.Db, item string) (*statsCollection, error) {
	sc, err := pu.initStatsForItem(db, item)
	if err == pudge.ErrKeyNotFound {
		dumpKeys(db)
		return nil, fmt.Errorf("no stats found for item %v", item)
	}
	return sc, err
}

func (pu *perItemStats) updateStats(db *pudge.Db, prefix string, item string, info *filewalk.PrefixInfo) error {
	sdb, err := pu.initStatsForItem(db, item)
	if err != nil && err != pudge.ErrKeyNotFound {
		return err
	}
	sdb.update(prefix, info)
	return nil
}

func (pu *perItemStats) remove(db *pudge.Db, prefix string, item string) error {
	sdb, err := pu.initStatsForItem(db, item)
	if err != nil && err != pudge.ErrKeyNotFound {
		return err
	}
	sdb.remove(prefix)
	return nil
}

func (pu *perItemStats) save(db *pudge.Db) error {
	// Take care to merge the set of users already loaded from the database
	// plus any that have actually been accessed.
	allItems := map[string]bool{}
	for _, u := range pu.itemKeys {
		allItems[u] = true
	}
	errs := errors.M{}
	for usr, stats := range pu.stats {
		if err := stats.save(db, usr); err != nil {
			errs.Append(err)
			continue
		}
		allItems[usr] = true
	}
	items := make([]string, 0, len(allItems))
	for k := range allItems {
		items = append(items, k)
	}
	sort.Strings(items)
	errs.Append(db.Set(pu.itemListKey, items))
	return errs.Err()
}
