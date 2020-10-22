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
	if err := db.Get(key, sc); err != nil {
		if err != pudge.ErrKeyNotFound {
			return err
		}
	}
	return nil
}

func (sc *statsCollection) save(db *pudge.Db, key string) error {
	return db.Set(key, sc)
}

func (sc *statsCollection) update(prefix string, info *filewalk.PrefixInfo) {
	sc.NumFiles.Update(prefix, int64(len(info.Files)))
	sc.NumChildren.Update(prefix, int64(len(info.Children)))
	sc.DiskUsage.Update(prefix, info.DiskUsage)
}

type perUserStats struct {
	stats map[string]*statsCollection
	users []string
}

func newPerUserStats() *perUserStats {
	return &perUserStats{
		stats: make(map[string]*statsCollection),
	}
}

func (pu *perUserStats) loadUserList(db *pudge.Db) error {
	if err := db.Get(usersListKey, &pu.users); err != nil && err != pudge.ErrKeyNotFound {
		return err
	}
	return nil
}

func (pu *perUserStats) initStatsForUser(db *pudge.Db, usr string) (*statsCollection, error) {
	sdb, ok := pu.stats[usr]
	if !ok {
		pu.stats[usr] = newStatsCollection(usr)
		err := db.Get(usr, pu.stats[usr])
		pu.users = append(pu.users, usr)
		return pu.stats[usr], err
	}
	return sdb, nil
}

func (pu *perUserStats) statsForUser(db *pudge.Db, usr string) (*statsCollection, error) {
	sc, err := pu.initStatsForUser(db, usr)
	if err == pudge.ErrKeyNotFound {
		return nil, fmt.Errorf("no stats found for user %v", usr)
	}
	return sc, err
}

func (pu *perUserStats) updateUserStats(db *pudge.Db, prefix string, info *filewalk.PrefixInfo) error {
	sdb, err := pu.initStatsForUser(db, info.UserID)
	if err != nil && err != pudge.ErrKeyNotFound {
		return err
	}
	sdb.update(prefix, info)
	return nil
}

func (pu *perUserStats) save(db *pudge.Db) error {
	// Take care to merge the set of users already loaded from the database
	// plus any that have actually been accessed.
	allUsers := map[string]bool{}
	for _, u := range pu.users {
		allUsers[u] = true
	}
	errs := errors.M{}
	for usr, stats := range pu.stats {
		if err := stats.save(db, usr); err != nil {
			errs.Append(err)
			continue
		}
		allUsers[usr] = true
	}
	users := make([]string, 0, len(allUsers))
	for k := range allUsers {
		users = append(users, k)
	}
	sort.Strings(users)
	errs.Append(db.Set(usersListKey, users))
	return errs.Err()
}
