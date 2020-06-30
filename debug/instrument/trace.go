// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	traceID int64
)

type trace struct {
	mu         sync.Mutex
	rootID, id int64
	records    []*record // records for a single goroutine
	gocaller   []uintptr
}

func appendRecord(t *trace, r *record) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.id == 0 {
		t.id = atomic.AddInt64(&traceID, 1)
		t.rootID = t.id
	}
	t.records = append(t.records, r)
}

func appendGoroutineTrace(parent, branch *trace, r *record) {
	branch.id = atomic.AddInt64(&traceID, 1)
	branch.rootID = parent.rootID
	branch.gocaller = r.callers
	r.goroutines = append(r.goroutines, branch)
	r.gocall = true
	parent.mu.Lock()
	defer parent.mu.Unlock()
	parent.records = append(parent.records, r)
}

// record is shared by CallTrace, MessageTrace etc.
type record struct {
	// locking is provided by at the trace level.
	callers    []uintptr
	gocall     bool
	time       time.Time
	arguments  interface{}
	payload    interface{}
	goroutines []*trace
}

func newRecord(skip int, args []interface{}) *record {
	var buf [64]uintptr
	n := runtime.Callers(skip, buf[:])
	pcs := make([]uintptr, n)
	copy(pcs, buf[:n])
	var nargs interface{}
	if len(args) == 1 {
		nargs = args[0]
	} else {
		tmp := make([]interface{}, len(args))
		copy(tmp, args)
		nargs = tmp
	}
	return &record{
		callers:   pcs,
		time:      time.Now(),
		arguments: nargs,
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
	if len(pcs) == 0 {
		return nil
	}
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

// WriteFrames writes out the supplied []runtime.Frame a frame per line
// prefixed by the supplied string.
func WriteFrames(out io.Writer, prefix string, frames []runtime.Frame) {
	for _, frame := range frames {
		fmt.Fprintf(out, "%s%s %s:%v\n",
			prefix,
			frame.Function,
			filepath.Base(frame.File),
			frame.Line)
	}
}

type walkRecord struct {
	arguments                interface{}
	payload                  interface{}
	id, rootID               int64
	time                     time.Time
	gocall                   bool
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
			id:        tr.id,
			rootID:    tr.rootID,
			gocall:    record.gocall,
			time:      record.time,
			arguments: record.arguments,
			payload:   record.payload,
			gocaller:  goframes,
			full:      frames,
			relative:  displayFrames,
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

func releaseArguments(tr *trace) {
	for i, record := range tr.records {
		tr.records[i].arguments = nil
		for _, goroutine := range record.goroutines {
			releaseArguments(goroutine)
		}
	}
}

func printArgs(args interface{}) string {
	if sl, ok := args.([]interface{}); ok {
		out := &strings.Builder{}
		out.WriteRune(' ')
		for i, v := range sl {
			out.WriteString(fmt.Sprintf("%v", v))
			if i < len(sl)-1 {
				out.WriteString(", ")
			}
		}
		return out.String()
	}
	if args == nil {
		return ""
	}
	return fmt.Sprintf(" %v", args)
}
