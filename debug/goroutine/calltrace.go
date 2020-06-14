// Package goroutine provides support for debugging goroutine related
// problems.
package goroutine

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// CallTrace represents a goroutine aware call trace where each record
// in the trace records the location it is called from. The trace can span
// goroutines via its Go method.

// records representing the execution of a given goroutine, each of which
// may be branch into multiple goroutines via the
type CallTrace struct {
	next, prev, last *CallTrace
	callers          []uintptr
	annotation       string
	parent           *CallTrace
	children         []*CallTrace
}

// MaxCallers represents the maximum number of PCs that can be recorded.
var MaxCallers = 32

// Record logs the current call site in the trace.
func (ct *CallTrace) Record(annotation string) {
	ct.add(annotation)
}

// Go logs the current call site and returns a new CallTrace, that is
// a child of the existing one, to be used in a goroutine started
// from the current one.
func (ct *CallTrace) Go(annotation string) *CallTrace {
	ct.add(annotation)
	nct := &CallTrace{parent: ct}
	nct.add(annotation)
	last := ct
	if ct.last != nil {
		last = ct.last
	}
	last.children = append(last.children, nct)
	return nct
}

func (ct *CallTrace) add(annotation string) {
	pcs := make([]uintptr, MaxCallers)
	n := runtime.Callers(3, pcs)
	pcs = pcs[:n]
	if ct.callers == nil {
		ct.callers = pcs
		ct.annotation = annotation
		return
	}
	nct := &CallTrace{callers: pcs, annotation: annotation}
	if ct.next == nil {
		ct.next = nct
	}
	nct.prev = nct.last
	if ct.last != nil {
		ct.last.next = nct
	}
	ct.last = nct
}

func (ct *CallTrace) String() string {
	out := &strings.Builder{}
	ct.string(0, nil, out, false)
	return out.String()
}

func (ct *CallTrace) DebugString() string {
	out := &strings.Builder{}
	ct.string(0, nil, out, true)
	return out.String()
}

func (ct *CallTrace) string(indent int, prev []runtime.Frame, out *strings.Builder, detailed bool) {
	spaces := strings.Repeat(" ", indent)
	cur := ct
	for {
		frames := framesFromPCs(cur.callers)
		reverseFrames(frames)
		displayFrames := trimPrefix(frames, prev)
		prev = frames
		if cur.parent == nil {
			if detailed {
				out.WriteString("\n")
			}
			fmt.Fprintf(out, "%s%s\n", spaces, cur.annotation)
		}
		if detailed {
			printFrames(spaces+"  ", displayFrames, out)
		}
		for _, child := range cur.children {
			child.string(indent+2, prev, out, detailed)
		}
		if cur = cur.next; cur == nil {
			break
		}
	}
	return
}

func trimPrefix(frames, prefix []runtime.Frame) []runtime.Frame {
	cp := commonPrefix(prefix, frames)
	if prefix != nil && cp > 0 {
		return frames[cp:]
	}
	return frames
}

func commonPrefix(a, b []runtime.Frame) int {
	l := min(len(a), len(b))
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

func printFrames(spaces string, frames []runtime.Frame, out *strings.Builder) {
	for _, frame := range frames {
		fmt.Fprintf(out, "%s%s %s:%v\n", spaces, frame.Function, filepath.Base(frame.File), frame.Line)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
