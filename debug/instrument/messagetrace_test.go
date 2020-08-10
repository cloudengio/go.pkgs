// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/debug/instrument"
)

var (
	localAddr, remoteAddr *net.IPAddr
)

func init() {
	localAddr, remoteAddr = &net.IPAddr{}, &net.IPAddr{}
	localAddr.IP = net.ParseIP("172.16.1.1")
	remoteAddr.IP = net.ParseIP("172.16.1.2")
}

func dumpMessageTrace(mt *instrument.MessageTrace) string {
	out := &strings.Builder{}
	mt.Print(out, true, true)
	return out.String()
}

func ExampleMessageTrace() {
	mt := &instrument.MessageTrace{}
	mt.Log(1, instrument.MessageSent, localAddr, remoteAddr, "some detail")
	mt.Log(1, instrument.MessageReceived, localAddr, remoteAddr, "some detail")

	fmt.Println(mt.String())
	mt.Print(os.Stdout, true, true)
}

func TestMessageTraceSimple(t *testing.T) {
	mt := &instrument.MessageTrace{}
	mt.Log(1, instrument.MessageSent, localAddr, remoteAddr, "sent something")
	mt.Log(1, instrument.MessageReceived, localAddr, remoteAddr, "received something")
	mt.Logf(1, instrument.MessageWait, localAddr, remoteAddr, "waiting for something")

	if got, want := sanitizeString(mt.String()), `  172.16.1.1 -> 172.16.1.2: sent something
  172.16.1.1 <- 172.16.1.2: received something
  172.16.1.1 <? 172.16.1.2: waiting for something
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeString(dumpMessageTrace(mt)), `  172.16.1.1 -> 172.16.1.2: sent something
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestMessageTraceSimple messagetrace_test.go:45

  172.16.1.1 <- 172.16.1.2: received something
    cloudeng.io/debug/instrument_test.TestMessageTraceSimple messagetrace_test.go:46

  172.16.1.1 <? 172.16.1.2: waiting for something
    cloudeng.io/debug/instrument_test.TestMessageTraceSimple messagetrace_test.go:47

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func generateMessageTrace() *instrument.MessageTrace {
	mt := &instrument.MessageTrace{}
	mt.Log(1, instrument.MessageSent, localAddr, remoteAddr, "first")
	var wg1, wg2 sync.WaitGroup
	n, m := 2, 2
	wg1.Add(n)
	wg2.Add(n * m)
	for i := 0; i < n; i++ {
		mt := mt.GoLogf(1, "launch goroutine 1")
		go func() {
			wg1.Done()
			mt = mt.GoLogf(1, "launch goroutine 2")
			for j := 0; j < m; j++ {
				go func() {
					mt.Log(1, instrument.MessageWait, localAddr, remoteAddr, "waiting")
					wg2.Done()
				}()
			}
		}()
	}
	wg1.Wait()
	wg2.Wait()
	return mt
}

func TestMessageTraceGoroutines(t *testing.T) {
	mt := generateMessageTrace()
	if got, want := sanitizeString(mt.String()), `  172.16.1.1 -> 172.16.1.2: first
  GoLog launch goroutine 1
    GoLog launch goroutine 2
      172.16.1.1 <? 172.16.1.2: waiting
      172.16.1.1 <? 172.16.1.2: waiting
  GoLog launch goroutine 1
    GoLog launch goroutine 2
      172.16.1.1 <? 172.16.1.2: waiting
      172.16.1.1 <? 172.16.1.2: waiting
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeString(dumpMessageTrace(mt)), `  172.16.1.1 -> 172.16.1.2: first
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestMessageTraceGoroutines messagetrace_test.go:97
    cloudeng.io/debug/instrument_test.generateMessageTrace messagetrace_test.go:73

  GoLog launch goroutine 1
    cloudeng.io/debug/instrument_test.generateMessageTrace messagetrace_test.go:79

    GoLog launch goroutine 2
      go @ cloudeng.io/debug/instrument_test.generateMessageTrace messagetrace_test.go:79
      cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82

      172.16.1.1 <? 172.16.1.2: waiting
        go @ cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82
        cloudeng.io/debug/instrument_test.generateMessageTrace.func1.1 messagetrace_test.go:85

      172.16.1.1 <? 172.16.1.2: waiting
        go @ cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82

  GoLog launch goroutine 1
    testing.tRunner testing.go:XXX
    cloudeng.io/debug/instrument_test.TestMessageTraceGoroutines messagetrace_test.go:97
    cloudeng.io/debug/instrument_test.generateMessageTrace messagetrace_test.go:79

    GoLog launch goroutine 2
      go @ cloudeng.io/debug/instrument_test.generateMessageTrace messagetrace_test.go:79
      cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82

      172.16.1.1 <? 172.16.1.2: waiting
        go @ cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82
        cloudeng.io/debug/instrument_test.generateMessageTrace.func1.1 messagetrace_test.go:85

      172.16.1.1 <? 172.16.1.2: waiting
        go @ cloudeng.io/debug/instrument_test.generateMessageTrace.func1 messagetrace_test.go:82

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if mt.ID() == 0 {
		t.Errorf("got zero for ID()")
	}
	if mt.RootID() == 0 {
		t.Errorf("got zero for RootID()")
	}
}

func TestMessageTraceRelease(t *testing.T) {
	mt := &instrument.MessageTrace{}
	mt.Log(1, instrument.MessageSent, localAddr, remoteAddr, 1, 2, 3, 4)
	id1, pid1 := mt.ID(), mt.RootID()
	mt.Log(1, instrument.MessageReceived, localAddr, remoteAddr, 5, 6, 7, 8)
	id2, pid2 := mt.ID(), mt.RootID()
	gmt := mt.GoLog(10, 11, 12, 13)
	gmt.Log(1, instrument.MessageAcceptWait, localAddr, remoteAddr, 100, 101)
	gmt.Log(1, instrument.MessageAccepted, localAddr, remoteAddr, 200, 201)

	if got, want := id1, id2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := pid1, pid2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := gmt.RootID(), mt.RootID(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := sanitizeString(mt.String()), `  172.16.1.1 -> 172.16.1.2: 1, 2, 3, 4
  172.16.1.1 <- 172.16.1.2: 5, 6, 7, 8
  GoLog 11, 12, 13
    172.16.1.1 <> 172.16.1.2: 100, 101
    172.16.1.1 == 172.16.1.2: 200, 201
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	mt.ReleaseArguments()
	if got, want := sanitizeString(mt.String()), `  172.16.1.1 -> 172.16.1.2:
  172.16.1.1 <- 172.16.1.2:
  GoLog
    172.16.1.1 <> 172.16.1.2:
    172.16.1.1 == 172.16.1.2:
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
