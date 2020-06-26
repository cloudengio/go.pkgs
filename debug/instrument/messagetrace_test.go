package goroutine_test

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"cloudeng.io/debug/goroutine"
)

var (
	localAddr, remoteAddr *net.IPAddr
)

func init() {
	localAddr, remoteAddr = &net.IPAddr{}, &net.IPAddr{}
	localAddr.IP = net.ParseIP("172.16.1.1")
	remoteAddr.IP = net.ParseIP("172.16.1.2")
}

func ExampleMessageTrace() {
	mt := &goroutine.MessageTrace{}
	mt.Log(1, goroutine.MessageSent, localAddr, remoteAddr, "some detail")
	mt.Log(1, goroutine.MessageReceived, localAddr, remoteAddr, "some detail")

	fmt.Printf(mt.String())
	fmt.Println(mt.Dump())
}

func TestMessageTraceSimple(t *testing.T) {
	mt := &goroutine.MessageTrace{}
	mt.Log(1, goroutine.MessageSent, localAddr, remoteAddr, "sent something")
	mt.Log(1, goroutine.MessageReceived, localAddr, remoteAddr, "received something")
	mt.Log(1, goroutine.MessageWait, localAddr, remoteAddr, "waiting for something")

	if got, want := sanitizeString(mt.String()), `  172.16.1.1 -> 172.16.1.2: sent something
  172.16.1.1 <- 172.16.1.2: received something
  172.16.1.1 <? 172.16.1.2: waiting for something
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeDump(mt.Dump()), `
  172.16.1.1 -> 172.16.1.2: sent something
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestMessageTraceSimple messagetrace_test.go:33

  172.16.1.1 <- 172.16.1.2: received something
    cloudeng.io/debug/goroutine_test.TestMessageTraceSimple messagetrace_test.go:34

  172.16.1.1 <? 172.16.1.2: waiting for something
    cloudeng.io/debug/goroutine_test.TestMessageTraceSimple messagetrace_test.go:35
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func generateMessageTrace() *goroutine.MessageTrace {
	mt := &goroutine.MessageTrace{}
	mt.Log(1, goroutine.MessageSent, localAddr, remoteAddr, "first")
	var wg1, wg2 sync.WaitGroup
	n, m := 2, 2
	wg1.Add(n)
	wg2.Add(n * m)
	for i := 0; i < n; i++ {
		mt := mt.Go(1)
		go func() {
			wg1.Done()
			mt = mt.Go(1)
			for j := 0; j < m; j++ {
				go func() {
					mt.Log(1, goroutine.MessageWait, localAddr, remoteAddr, "waiting")
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
  go func()....
    go func()....
      172.16.1.1 <? 172.16.1.2: waiting
      172.16.1.1 <? 172.16.1.2: waiting
  go func()....
    go func()....
      172.16.1.1 <? 172.16.1.2: waiting
      172.16.1.1 <? 172.16.1.2: waiting
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeDump(mt.Dump()), `
  172.16.1.1 -> 172.16.1.2: first
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestMessageTraceGoroutines messagetrace_test.go:85
    cloudeng.io/debug/goroutine_test.generateMessageTrace messagetrace_test.go:61

  go func()....
    cloudeng.io/debug/goroutine_test.generateMessageTrace messagetrace_test.go:67

    go func()....
      cloudeng.io/debug/goroutine_test.generateMessageTrace.func1 messagetrace_test.go:70

      172.16.1.1 <? 172.16.1.2: waiting
        cloudeng.io/debug/goroutine_test.generateMessageTrace.func1.1 messagetrace_test.go:73

      172.16.1.1 <? 172.16.1.2: waiting

  go func()....
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestMessageTraceGoroutines messagetrace_test.go:85
    cloudeng.io/debug/goroutine_test.generateMessageTrace messagetrace_test.go:67

    go func()....
      cloudeng.io/debug/goroutine_test.generateMessageTrace.func1 messagetrace_test.go:70

      172.16.1.1 <? 172.16.1.2: waiting
        cloudeng.io/debug/goroutine_test.generateMessageTrace.func1.1 messagetrace_test.go:73

      172.16.1.1 <? 172.16.1.2: waiting
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
