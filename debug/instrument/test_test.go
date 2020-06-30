// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument

import (
	"time"
)

// SetTime sets the timestamp for all records in this trace, it's
// for testing purposes only.
func SetTime(mr *MessageTrace, when time.Time) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for _, m := range mr.records {
		m.time = when
	}
}
