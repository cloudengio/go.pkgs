package goroutine

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