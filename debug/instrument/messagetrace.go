package goroutine

import (
	"fmt"
	"net"
	"runtime"
	"sort"
	"strings"
	"time"
)

type MessageTrace struct {
	trace
}

// ID returns the id of this message trace. All traces are allocated a unique
// id on first use, otherwise their id is zero.
func (mt *MessageTrace) ID() int64 {
	return mt.id
}

// ParentID returns the parent id of this message trace, that is the id
// that is allocated to the first MessageTrace record in this call trace.
func (mt *MessageTrace) ParentID() int64 {
	return mt.parentID
}

type MessageStatus int

const (
	MessageWait MessageStatus = iota + 1
	MessageSent
	MessageReceived
	MessageAcceptWait
	MessageAccepted
)

func (m MessageStatus) String() string {
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
	status        MessageStatus
	local, remote net.Addr
	detail        interface{}
}

func (mt *MessageTrace) Log(skip int, status MessageStatus, local, remote net.Addr, detail interface{}) {
	record := newRecord(skip + 2)
	record.payload = messageRecord{
		status: status,
		local:  local,
		remote: remote,
		detail: detail,
	}
	appendRecord(&mt.trace, record)
}

func (mt *MessageTrace) Go(skip int) *MessageTrace {
	record := newRecord(skip + 2)
	record.payload = messageRecord{}
	nct := &MessageTrace{}
	appendGoroutineTrace(&mt.trace, &nct.trace, record)
	return nct
}

func (mt *MessageTrace) String() string {
	out := &strings.Builder{}
	mt.string(out, false)
	return out.String()
}

func (mt *MessageTrace) Dump() string {
	out := &strings.Builder{}
	fmt.Fprintf(out, "message trace % 8d : begin ----------------------\n", mt.id)
	mt.string(out, true)
	fmt.Fprintf(out, "message trace % 8d : end   ----------------------\n", mt.id)
	return out.String()
}

func (mt *MessageTrace) string(out *strings.Builder, detailed bool) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	walk(&mt.trace, 0, nil, func(level int, wr *walkRecord) {
		payload := wr.payload.(messageRecord)
		spaces := strings.Repeat(" ", (level+1)*2)
		if detailed {
			out.WriteString("\n")
		}
		fmt.Fprintf(out, "%s(%s:% 6d/%d) ",
			spaces, wr.time.Format("0102 15:04:05.000000"), wr.parentID, wr.id)
		if payload.status == 0 {
			out.WriteString("go func()....\n")
		} else {
			fmt.Fprintf(out, "%s %s %s: %s\n",
				payload.local, payload.status, payload.remote, payload.detail)
		}
		if detailed {
			printFrames(spaces+"  ", wr.relative, out)
		}
	})
}

// MessageRecord represents the metadata for a recorded message.
type MessageRecord struct {
	Name          string          // Name assigned to this message trace by Flatten.
	ID, ParentID  int64           // The ID and ParentID of this record in the original trace.
	Time          time.Time       // The time that the message was logged.
	Status        MessageStatus   // The status of the message.
	Local, Remote net.Addr        // The local and remote addresses for the message.
	Detail        interface{}     // The detail logged with the message.
	GoCaller      []runtime.Frame // The full call stack of where the goroutine was launced from.
	Callers       []runtime.Frame // The full call stack.
}

type MessageRecords []MessageRecord

// Flatten returns a slice of MessageRecords sorted primarily by time and
// then by message status (in order of Waiting, Sent and Received).
func (mt *MessageTrace) Flatten(name string) MessageRecords {
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
			Name:     name,
			ID:       wr.id,
			ParentID: wr.parentID,
			Time:     wr.time,
			Status:   payload.status,
			Local:    payload.local,
			Remote:   payload.remote,
			Detail:   payload.detail,
			GoCaller: wr.gocaller,
			Callers:  wr.full,
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
		if mr.Status == 0 {
			continue
		}
		fmt.Fprintf(out, "(%s:% 6d/%d) % 20s: %s %s %s: %s\n",
			mr.Time.Format("0102 15:04:05.000000"),
			mr.ParentID,
			mr.ID,
			mr.Name,
			mr.Local,
			mr.Status,
			mr.Remote,
			mr.Detail,
		)
	}
	return out.String()
}

func sorter(a, b MessageRecord) bool {
	switch {
	case a.Time.Before(b.Time):
		return true
	case a.Time.Equal(b.Time):
		switch {
		case a.Status == MessageWait:
			return true
		case b.Status == MessageWait:
			return false
		default:
			return a.Status == MessageSent
		}
	}
	return false
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

func MergeMesageTraces(traces ...MessageRecords) MessageRecords {
	switch len(traces) {
	case 0:
		return MessageRecords{}
	case 1:
		return traces[0]
	}
	merged := mergePair(traces[0], traces[1])
	traces = append([]MessageRecords{merged}, traces[2:]...)
	return MergeMesageTraces(traces...)
}
