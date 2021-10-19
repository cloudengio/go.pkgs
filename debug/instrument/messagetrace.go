// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
)

// MessageTrace provides the ability to log various communication primitives
// (e.g. message sent, received etc) and their location in a linear execution
// (Log, Logf) as well as to span the creation of goroutines and the execution
// of said primitives in their linear execution (GoLog, GoLogf). A log record
// consists of the parameters to the logging function and the location of the
// call (ie. caller stackframes).
type MessageTrace struct {
	trace
}

// ID returns the id of this message trace. All traces are allocated a unique
// id on first use, otherwise their id is zero.
func (mt *MessageTrace) ID() int64 {
	return mt.id
}

// RootID returns the root id of this message trace, that is the id
// that is allocated to the first MessageTrace record in this call trace.
func (mt *MessageTrace) RootID() int64 {
	return mt.rootID
}

// MessagePrimitive represents the supported message operations.
type MessagePrimitive int

// The above are the defined communication primitives. They are defined
// in order of preference when sorting by MergeMessageTraces.
const (
	MessageWait MessagePrimitive = iota + 1
	MessageAcceptWait
	MessageAccepted
	MessageSent
	MessageReceived
)

// String implements fmt.Stringer.
func (m MessagePrimitive) String() string {
	switch m {
	case MessageWait:
		return "<?"
	case MessageSent:
		return "->"
	case MessageReceived:
		return "<-"
	case MessageAcceptWait:
		return "<>"
	case MessageAccepted:
		return "=="
	}
	return "unrecognised status"
}

type messageRecord struct {
	status        MessagePrimitive
	local, remote net.Addr
}

// Log logs the current call site and its arguments. The supplied arguments
// are stored in a slice and retained until ReleaseArguments is called.
// Skip is the number of callers to skip, as per runtime.Callers.
func (mt *MessageTrace) Log(skip int, status MessagePrimitive, local, remote net.Addr, args ...interface{}) {
	record := newRecord(skip+2, args)
	record.payload = messageRecord{
		status: status,
		local:  local,
		remote: remote,
	}
	appendRecord(&mt.trace, record)
}

// ReleaseArguments releases all stored arguments from previous
// calls to Log or Logf.
func (mt *MessageTrace) ReleaseArguments() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	releaseArguments(&mt.trace)
}

// Logf logs the current call site with its arguments being immediately
// used to create a string (using fmt.Sprintf) that is stored within the trace.
// Skip is the number of callers to skip, as per runtime.Callers.
func (mt *MessageTrace) Logf(skip int, status MessagePrimitive, local, remote net.Addr, format string, args ...interface{}) {
	mt.Log(skip+1, status, local, remote, fmt.Sprintf(format, args...))
}

// GoLog logs the current call site and returns a new MessageTrace, that is
// a child of the existing one, to be used in a goroutine started from the
// current one. Skip is the number of callers to skip, as per runtime.Callers.
func (mt *MessageTrace) GoLog(skip int, args ...interface{}) *MessageTrace {
	record := newRecord(skip+2, args)
	record.payload = messageRecord{}
	nct := &MessageTrace{}
	appendGoroutineTrace(&mt.trace, &nct.trace, record)
	return nct
}

// GoLogf logs the current call site and returns a new MessageTrace, that is
// a child of the existing one, to be used in a goroutine started from the
// current one. Skip is the number of callers to skip, as per runtime.Callers.
func (mt *MessageTrace) GoLogf(skip int, format string, args ...interface{}) *MessageTrace {
	return mt.GoLog(skip+1, fmt.Sprintf(format, args...))
}

// Print will print the trace to the supplied io.Writer, if callers is set
// then the stack frame will be printed and if relative is set each displayed
// stack frame will be relative to the previous one for
func (mt *MessageTrace) Print(out io.Writer, callers, relative bool) {
	mt.Walk(func(mr MessageRecord) {
		printCallRecord(mr.String(), &mr.CallRecord, out, callers, relative)
	})
}

// String implements fmt.Stringer.
func (mt *MessageTrace) String() string {
	out := &strings.Builder{}
	mt.Walk(func(mr MessageRecord) {
		out.WriteString(strings.Repeat(" ", (mr.Level+1)*2))
		out.WriteString(mr.String())
		out.WriteString("\n")
	})
	return out.String()
}

// Walk traverses the call trace calling the supplied function for each record.
func (mt *MessageTrace) Walk(fn func(mr MessageRecord)) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	walk(&mt.trace, 0, nil, func(level int, wr *walkRecord) {
		payload := wr.payload.(messageRecord)
		fn(MessageRecord{
			CallRecord: newCallRecord(level, wr),
			Status:     payload.status,
			Local:      payload.local,
			Remote:     payload.remote,
		})
	})
}

// MessageRecord represents the metadata for a recorded message.
type MessageRecord struct {
	CallRecord
	Tag           string           // Tag assigned to this message trace by Flatten.
	Status        MessagePrimitive // The status of the message.
	Local, Remote net.Addr         // The local and remote addresses for the message.
}

// String implements fmt.Stringer.
func (mr MessageRecord) String() string {
	out := &strings.Builder{}
	out.WriteString(mr.prefix())
	if len(mr.Tag) > 0 {
		fmt.Fprintf(out, "% 20s:", mr.Tag)
	}
	if !mr.GoCall {
		fmt.Fprintf(out, " %s %s %s:",
			mr.Local,
			mr.Status,
			mr.Remote)
	}
	out.WriteString(printArgs(mr.Arguments))
	return out.String()
}

type MessageRecords []MessageRecord

// Flatten returns a slice of MessageRecords sorted by level, rootID, ID,
// time and finally by message status (in order of Waiting, Sent and Received).
func (mt *MessageTrace) Flatten(tag string) MessageRecords {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	out := make([]MessageRecord, 0, len(mt.records))
	walk(&mt.trace, 0, nil, func(level int, wr *walkRecord) {
		payload := wr.payload.(messageRecord)
		if payload.status == 0 {
			// skip the record created when a go routine is spawned.
			return
		}
		out = append(out, MessageRecord{
			CallRecord: newCallRecord(0, wr),
			Tag:        tag,
			Status:     payload.status,
			Local:      payload.local,
			Remote:     payload.remote,
		})
	})
	sort.Slice(out, func(i, j int) bool {
		return sorter(out[i], out[j])
	})
	return out
}

func (ms MessageRecords) String() string {
	out := &strings.Builder{}
	for _, mr := range ms {
		fmt.Fprintln(out, mr.String())
	}
	return out.String()
}

func sorter(a, b MessageRecord) bool {
	switch {
	case a.Level != b.Level:
		return a.Level < b.Level
	case a.RootID != b.RootID:
		return a.RootID < b.RootID
	case a.ID != b.ID:
		return a.ID < b.ID
	case !a.Time.Equal(b.Time):
		return a.Time.Before(b.Time)
	default:
		return a.Status < b.Status
	}
}

func mergePair(a, b MessageRecords) MessageRecords {
	var i, j int
	na, nb := len(a), len(b)
	out := make([]MessageRecord, 0, na+nb)
	for {
		if i >= na || j >= nb {
			break
		}
		if sorter(a[i], b[j]) {
			out = append(out, a[i])
			i++
			continue
		}
		out = append(out, b[j])
		j++
	}
	if i < na {
		out = append(out, a[i:]...)
	}
	if j < nb {
		out = append(out, b[j:]...)
	}
	return out
}

func MergeMessageTraces(traces ...MessageRecords) MessageRecords {
	switch len(traces) {
	case 0:
		return MessageRecords{}
	case 1:
		return traces[0]
	}
	merged := mergePair(traces[0], traces[1])
	traces = append([]MessageRecords{merged}, traces[2:]...)
	return MergeMessageTraces(traces...)
}
