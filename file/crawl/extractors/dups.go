// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package extractors

import "sync"

type dupmap struct {
	sync.Mutex
	crawled map[string]struct{}
}

func newDupmap() *dupmap {
	return &dupmap{crawled: map[string]struct{}{}}
}

func (dm *dupmap) setOrExists(name string) bool {
	if _, ok := dm.crawled[name]; ok {
		return true
	}
	dm.crawled[name] = struct{}{}
	return false
}
