// Package goroutine provides support for debugging goroutine related
// problems.
package goroutine

import (
	"fmt"
	"strings"
)

// CallTrace represents a goroutine aware call trace where each record
// in the trace records the location it is called from. The trace can span
// goroutines via its Go method.
// TODO: explain trace structure, parent and trace IDs, prefix removal etc.
// TODO: figure out a more flexible display/output option.
type CallTrace struct {
	trace
}

// ID returns the id of this calltrace. All traces are allocated a unique
// id on first use, otherwise their id is zero.
func (ct *CallTrace) ID() int64 {
	return ct.id
}

// ParentID returns the parent id of this calltrace, that is the id that is
// allocated to the first CallTrace record in this call trace hierarchy.
func (ct *CallTrace) ParentID() int64 {
	return ct.parentID
}

// Logf logs the current call site and message. Skip is the number of callers
// to skip, as per runtime.Callers.
func (ct *CallTrace) Logf(skip int, format string, args ...interface{}) {
	record := newRecord(skip + 2)
	record.payload = fmt.Sprintf(format, args...)
	appendRecord(&ct.trace, record)
}

// GoLogf logs the current call site and returns a new CallTrace, that is
// a child of the existing one, to be used in a goroutine started from the
// current one. Skip is the number of callers to skip, as per runtime.Callers.
func (ct *CallTrace) GoLogf(skip int, format string, args ...interface{}) *CallTrace {
	record := newRecord(skip + 2)
	record.payload = fmt.Sprintf(format, args...)
	nct := &CallTrace{}
	appendGoroutineTrace(&ct.trace, &nct.trace, record)
	return nct
}

func (ct *CallTrace) String() string {
	out := &strings.Builder{}
	ct.string(out, false)
	return out.String()
}

func (ct *CallTrace) Dump() string {
	out := &strings.Builder{}
	fmt.Fprintf(out, "call trace % 8d : begin ----------------------\n", ct.id)
	ct.string(out, true)
	fmt.Fprintf(out, "call trace % 8d : end   ----------------------\n", ct.id)
	return out.String()
}

func (ct *CallTrace) string(out *strings.Builder, detailed bool) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	walk(&ct.trace, 0, nil, func(level int, wr *walkRecord) {
		spaces := strings.Repeat(" ", (level+1)*2)
		if detailed {
			out.WriteString("\n")
		}
		fmt.Fprintf(out, "%s(%s:% 6d/%d) %s\n",
			spaces, wr.time.Format("0102 15:04:05.000000"), wr.parentID, wr.id, wr.payload.(string))
		if detailed {
			printFrames(spaces+"  ", wr.relative, out)
		}
	})
}
