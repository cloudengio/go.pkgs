// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/debug/instrument"
)

var (
	captureLeaderRE = regexp.MustCompile(`([ ]*)\(([^)]+)\)[ ]*(.*)`)
	idsRE           = regexp.MustCompile(`(\d+)/(\d+)`)
)

func sanitizeString(s string) string {
	out := &strings.Builder{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		l := sc.Text()
		parts := captureLeaderRE.FindStringSubmatch(l)
		if len(parts) == 4 {
			fmt.Fprintf(out, "%s%s\n", parts[1], parts[3])
			continue
		}
		out.WriteString(l)
		out.WriteString("\n")
	}
	return out.String()
}

type timeEtc struct {
	when     time.Time
	id       int64
	parentID int64
	args     string
}

func getTimeAndIDs(s string) ([]timeEtc, error) {
	recs := []timeEtc{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		l := sc.Text()
		parts := captureLeaderRE.FindStringSubmatch(l)
		if len(parts) != 4 {
			return nil, fmt.Errorf("failed to match line: %v", l)
		}
		tmp := parts[2][:26]
		when, err := time.Parse("060102 15:04:05.000000 MST", tmp)
		if err != nil {
			return nil, fmt.Errorf("malformed time: %v: %v", tmp, err)
		}
		tmp = parts[2][27:]
		idparts := idsRE.FindStringSubmatch(tmp)
		if len(idparts) != 3 {
			return nil, fmt.Errorf("failed to find ids in %v from line: %v", tmp, l)
		}
		id, err := strconv.ParseInt(idparts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id %v from line: %v", idparts[1], l)
		}
		parent, err := strconv.ParseInt(idparts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent id %v from line: %v", idparts[2], l)
		}
		recs = append(recs, timeEtc{
			id:       id,
			parentID: parent,
			when:     when,
			args:     parts[3],
		})
	}
	return recs, nil
}

func ExampleCallTrace() {
	ct := &instrument.CallTrace{}
	ct.Logf(1, "a")
	ct.Logf(1, "b")
	var wg sync.WaitGroup
	n := 2
	wg.Add(n)
	for i := 0; i < n; i++ {
		ct := ct.GoLogf(1, "goroutine launch")
		go func(i int) {
			ct.Logf(1, fmt.Sprintf("inside goroutine %v", i))
			wg.Done()
			ct.Logf(1, fmt.Sprintf("inside goroutine %v", i))
		}(i)
	}
	wg.Wait()
	// Print the call trace without stack frames.
	fmt.Println(ct.String())
	// Print the call trace with relative stack frames.
	ct.Print(os.Stdout, true, true)
}

func dumpCallTrace(ct *instrument.CallTrace) string {
	out := &strings.Builder{}
	ct.Print(out, true, true)
	return out.String()
}

func TestCallTraceSimple(t *testing.T) {
	ct := &instrument.CallTrace{}
	ct.Logf(1, "a")
	ct.Logf(1, "b")
	ct.Logf(1, "c")
	if got, want := sanitizeString(ct.String()), `  a
  b
  c
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeString(dumpCallTrace(ct)), `  a
    testing.tRunner testing.go:991
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:113

  b
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:114

  c
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:115

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	headers, err := getTimeAndIDs(ct.String())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(headers), 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	now := time.Now()
	for _, h := range headers {
		if h.when.After(now) {
			t.Errorf("timestamp is in the future: %v", h.when)
		} else if now.Sub(h.when) > time.Minute*5 {
			t.Errorf("timestamp is outside of a reasonable range: %v: ", h.when)
		}
		if got, want := h.id, int64(1); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := h.parentID, int64(1); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestCallTraceGoroutines(t *testing.T) {
	ct := &instrument.CallTrace{}
	ct.Logf(1, "a")
	var wg1, wg2 sync.WaitGroup
	n, m := 2, 2
	wg1.Add(n)
	wg2.Add(n * m)
	for i := 0; i < n; i++ {
		ct := ct.GoLogf(1, "goroutine L1 launch")
		go func() {
			ct.Logf(1, "inside L1 goroutine")
			wg1.Done()
			ct.Logf(1, "inside L1 goroutine")
			ct = ct.GoLogf(1, "goroutine L2 launch")
			for j := 0; j < m; j++ {
				go func() {
					ct.Logf(1, "inside L2 goroutine")
					wg2.Done()
				}()
			}
		}()
	}
	wg1.Wait()
	wg2.Wait()
	if got, want := sanitizeString(ct.String()), `  a
  GoLog goroutine L1 launch
    inside L1 goroutine
    inside L1 goroutine
    GoLog goroutine L2 launch
      inside L2 goroutine
      inside L2 goroutine
  GoLog goroutine L1 launch
    inside L1 goroutine
    inside L1 goroutine
    GoLog goroutine L2 launch
      inside L2 goroutine
      inside L2 goroutine
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeString(dumpCallTrace(ct)), `  a
    testing.tRunner testing.go:991
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:162

  GoLog goroutine L1 launch
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:170

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:172

    GoLog goroutine L2 launch
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173
        cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:176

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173

  GoLog goroutine L1 launch
    testing.tRunner testing.go:991
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:170

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:172

    GoLog goroutine L2 launch
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:168
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173
        cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:176

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:173

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if ct.ID() == 0 {
		t.Errorf("got zero for ID()")
	}
	if ct.RootID() == 0 {
		t.Errorf("got zero for RootID()")
	}
}

func TestCallTraceRelease(t *testing.T) {
	ct := &instrument.CallTrace{}
	ct.Log(1, 1, 2, 3, 4)
	id1, pid1 := ct.ID(), ct.RootID()
	ct.Log(1, 5, 6, 7, 8)
	id2, pid2 := ct.ID(), ct.RootID()
	gct := ct.GoLog(10, 11, 12, 13)
	gct.Log(1, 100, 101)
	gct.Log(1, 200, 201)

	if got, want := id1, id2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := pid1, pid2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := gct.RootID(), ct.ID(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := sanitizeString(ct.String()), `  1, 2, 3, 4
  5, 6, 7, 8
  GoLog 11, 12, 13
    100, 101
    200, 201
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	ct.ReleaseArguments()
	if got, want := sanitizeString(ct.String()), `  
  
  GoLog
    
    
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
