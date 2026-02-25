// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/debug/instrument"
)

func ExampleCallTrace() {
	ct := &instrument.CallTrace{}
	ct.Logf(1, "a")
	ct.Logf(1, "b")
	var wg sync.WaitGroup
	n := 2
	wg.Add(n)
	for i := range n {
		ct := ct.GoLogf(1, "goroutine launch")
		go func(i int) {
			ct.Logf(1, "%s", fmt.Sprintf("inside goroutine %v", i))
			wg.Done()
			ct.Logf(1, "%s", fmt.Sprintf("inside goroutine %v", i))
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
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:48

  b
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:49

  c
    cloudeng.io/debug/instrument_test.TestCallTraceSimple calltrace_test.go:50

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
	for range n {
		ct := ct.GoLogf(1, "goroutine L1 launch")
		go func() {
			ct.Logf(1, "inside L1 goroutine")
			wg1.Done()
			ct.Logf(1, "inside L1 goroutine")
			ct = ct.GoLogf(1, "goroutine L2 launch")
			for range m {
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
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:97

  GoLog goroutine L1 launch
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:105

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:107

    GoLog goroutine L2 launch
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108
        cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:111

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108

  GoLog goroutine L1 launch
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:105

    inside L1 goroutine
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:107

    GoLog goroutine L2 launch
      go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines calltrace_test.go:103
      cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108
        cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:111

      inside L2 goroutine
        go @ cloudeng.io/debug/instrument_test.TestCallTraceGoroutines.func1 calltrace_test.go:108

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
