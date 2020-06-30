// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package instrument provides support for instrumenting complex applications
// to trace their operation and communication behaviour across multiple
// goroutines and contexts.
//
// The underlying data structure used for the trace related instrumentation
// tools is a combination of a linked list of trace records where
// each individual record in that linked list can itself host any number
// linked lists of trace records. This mimics a linear execution which
// can spawn concurrent traces from any point, that is, a linear execution
// which spawns goroutines. The various uses of this underlying structure
// offer a 'Go' method to be used in conjunction with goroutines to allow
// the traces to span multiple goroutines.
//
// Functions are provided to register and retrieve traces from context.Context
// instances and thus to pass them through multiple API boundaries.
package instrument

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"
)

// CallTrace provides the ability to log specific points in a linear execution
// (Log, Logf) as well as to span the creation of goroutines and points
// in their execution (GoLog, GoLogf). A log record consists of the
// parameters to the logging function and the location of the call
// (ie. caller stackframes).
type CallTrace struct {
	trace
}

// ID returns the id of this call trace. All traces are allocated a unique
// id on first use, otherwise their id is zero.
func (ct *CallTrace) ID() int64 {
	return ct.id
}

// RootID returns the root id of this call trace, ie. the id that was
// allocated to the first record in this call trace hierarchy.
func (ct *CallTrace) RootID() int64 {
	return ct.rootID
}

// Log logs the current call site and its arguments. The supplied arguments
// are stored in a slice and retained until ReleaseArguments is called.
// Skip is the number of callers to skip, as per runtime.Callers.
func (ct *CallTrace) Log(skip int, args ...interface{}) {
	record := newRecord(skip+2, args)
	appendRecord(&ct.trace, record)
}

// ReleaseArguments releases all stored arguments from previous
// calls to Log or Logf.
func (ct *CallTrace) ReleaseArguments() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	releaseArguments(&ct.trace)
}

// Logf logs the current call site with its arguments being immediately
// used to create a string (using fmt.Sprintf) that is stored within the trace.
// Skip is the number of callers to skip, as per runtime.Callers.
func (ct *CallTrace) Logf(skip int, format string, args ...interface{}) {
	ct.Log(skip+1, fmt.Sprintf(format, args...))
}

// GoLog logs the current call site and returns a new CallTrace, that is
// a child of the existing one, to be used in a goroutine started from the
// current one. Skip is the number of callers to skip, as per runtime.Callers.
func (ct *CallTrace) GoLog(skip int, args ...interface{}) *CallTrace {
	record := newRecord(skip+2, args)
	nct := &CallTrace{}
	appendGoroutineTrace(&ct.trace, &nct.trace, record)
	return nct
}

// GoLogf logs the current call site and returns a new CallTrace, that is
// a child of the existing one, to be used in a goroutine started from the
// current one. Skip is the number of callers to skip, as per runtime.Callers.
func (ct *CallTrace) GoLogf(skip int, format string, args ...interface{}) *CallTrace {
	return ct.GoLog(skip+1, fmt.Sprintf(format, args...))
}

// Print will print the trace to the supplied io.Writer, if callers is set
// then the stack frame will be printed and if relative is set each displayed
// stack frame will be relative to the previous one for
func (ct *CallTrace) Print(out io.Writer, callers, relative bool) {
	ct.Walk(func(cr CallRecord) {
		printCallRecord(cr.String(), &cr, out, callers, relative)
	})
}

// String implements fmt.Stringer.
func (ct *CallTrace) String() string {
	out := &strings.Builder{}
	ct.Print(out, false, false)
	return out.String()
}

// CallRecord represents a recorded trace location.
type CallRecord struct {
	// ID is the id of the current trace, and RootID the ID of the
	// trace that created this one via a GoLog or GoLogf call.
	ID, RootID int64
	// Level is the number of GoLog or GoLogf calls that preceded
	// the creation of this record.
	Level int
	// Time is the time that the record was created at.
	Time time.Time
	// GoCall is true if this record was generated by a GoLo or GoLogf
	// call.
	GoCall bool
	// Full is the full stack frame of recorded location, whereas Relative
	// is relative to the previous recorded location.
	Full, Relative []runtime.Frame
	// GoCaller is the full stack frame of the call to GoLog or GoLogf
	// that created this sub-trace.
	GoCaller []runtime.Frame
	// Arguments is either the formatted string for Logf or a slice
	// containing the arguments to Log.
	Arguments interface{}
}

func (cr *CallRecord) prefix() string {
	isgo := ""
	if cr.GoCall {
		isgo = " GoLog"
	}
	return fmt.Sprintf("(%s:% 6d/%d)%s",
		cr.Time.Format("060102 15:04:05.000000 MST"), cr.RootID, cr.ID, isgo)
}

// String implements fmt.Stringer.
func (cr *CallRecord) String() string {
	out := &strings.Builder{}
	out.WriteString(cr.prefix())
	out.WriteString(printArgs(cr.Arguments))
	return out.String()
}

// Walk traverses the call trace calling the supplied function for each record.
func (ct *CallTrace) Walk(fn func(cr CallRecord)) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	walk(&ct.trace, 0, nil, func(level int, wr *walkRecord) {
		fn(newCallRecord(level, wr))
	})
}

func newCallRecord(level int, wr *walkRecord) CallRecord {
	return CallRecord{
		Level:     level,
		Time:      wr.time,
		GoCall:    wr.gocall,
		ID:        wr.id,
		RootID:    wr.rootID,
		Arguments: wr.arguments,
		GoCaller:  wr.gocaller,
		Full:      wr.full,
		Relative:  wr.relative,
	}
}

func printCallRecord(summary string, cr *CallRecord, out io.Writer, callers, relative bool) {
	indent := strings.Repeat(" ", (cr.Level+1)*2)
	fmt.Fprint(out, indent, summary)
	if callers {
		out.Write([]byte{'\n'})
		indent += "  "
		if len(cr.GoCaller) > 0 {
			goindent := indent + "go @ "
			WriteFrames(out, goindent, cr.GoCaller[:1])
		}
		frames := cr.Full
		if relative {
			frames = cr.Relative
		}
		WriteFrames(out, indent, frames)
	}
	out.Write([]byte{'\n'})
}
