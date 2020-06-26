package goroutine

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	traceID       int64
	traceParentID int64
)

type trace struct {
	mu           sync.Mutex
	parentID, id int64
	records      []*record // records for a single goroutine
	gocaller     []uintptr
}

func appendRecord(t *trace, r *record) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.id == 0 {
		t.id = atomic.AddInt64(&traceID, 1)
		t.parentID = atomic.AddInt64(&traceParentID, 1)
	}
	t.records = append(t.records, r)
}

func appendGoroutineTrace(parent, branch *trace, r *record) {
	branch.id = atomic.AddInt64(&traceID, 1)
	branch.parentID = parent.parentID
	r.goroutines = append(r.goroutines, branch)
	parent.mu.Lock()
	defer parent.mu.Unlock()
	parent.records = append(parent.records, r)
	parent.gocaller = r.callers
}

// record is shared by CallTrace, MessageTrace etc.
type record struct {
	// locking is provided by at the trace level.
	callers    []uintptr
	time       time.Time
	payload    interface{}
	goroutines []*trace
}

func newRecord(skip int) *record {
	var buf [64]uintptr
	n := runtime.Callers(skip, buf[:])
	pcs := make([]uintptr, n)
	copy(pcs, buf[:n])
	return &record{
		callers: pcs,
		time:    time.Now(),
	}
}

func trimPrefix(frames, prefix []runtime.Frame) []runtime.Frame {
	cp := commonPrefix(prefix, frames)
	if prefix != nil && cp > 0 {
		return frames[cp:]
	}
	return frames
}

func commonPrefix(a, b []runtime.Frame) int {
	l := len(a)
	if bl := len(b); bl < l {
		l = len(b)
	}
	for i := 0; i < l; i++ {
		if a[i].PC != b[i].PC {
			return i
		}
	}
	return l
}

func reverseFrames(frames []runtime.Frame) {
	l := len(frames)
	for i := 0; i < l/2; i++ {
		frames[i], frames[l-1] = frames[l-1], frames[i]
		l--
	}
}

func framesFromPCs(pcs []uintptr) []runtime.Frame {
	out := make([]runtime.Frame, len(pcs))
	frames := runtime.CallersFrames(pcs)
	i := 0
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		out[i] = frame
		i++
	}
	return out[:i]
}

func printFrames(spaces string, frames []runtime.Frame, out io.Writer) {
	for _, frame := range frames {
		fmt.Fprintf(out, "%s%s %s:%v\n", spaces, frame.Function, filepath.Base(frame.File), frame.Line)
	}
}

type walkRecord struct {
	payload                  interface{}
	id, parentID             int64
	time                     time.Time
	gocaller, full, relative []runtime.Frame
}

func walk(tr *trace, level int, prev []runtime.Frame, fn func(level int, wr *walkRecord)) {
	for _, record := range tr.records {
		frames := framesFromPCs(record.callers)
		goframes := framesFromPCs(tr.gocaller)
		reverseFrames(frames)
		displayFrames := trimPrefix(frames, prev)
		prev = frames
		fn(level, &walkRecord{
			payload:  record.payload,
			id:       tr.id,
			parentID: tr.parentID,
			time:     record.time,
			gocaller: goframes,
			full:     frames,
			relative: displayFrames,
		})
		for _, goroutine := range record.goroutines {
			walk(goroutine, level+1, prev, fn)
		}
		if len(record.goroutines) > 0 {
			// Display the full stack frame after indented display
			// of goroutine information.
			prev = nil
		}
	}
}
