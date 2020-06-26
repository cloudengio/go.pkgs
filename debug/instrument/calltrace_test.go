package goroutine_test

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/debug/goroutine"
)

var captureLeaderRE = regexp.MustCompile(`([ ]*)\(([^)]+)\)[ ]*(.*)`)

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

func sanitizeDump(s string) string {
	out := &strings.Builder{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "begin --------") ||
			strings.Contains(l, "end   -------") {
			continue
		}
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

func ExampleCallTrace() {
	ct := &goroutine.CallTrace{}
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
	fmt.Println(ct.String())
	fmt.Println(ct.Dump())
}

func TestCallTraceSimple(t *testing.T) {
	ct := &goroutine.CallTrace{}
	ct.Logf(1, "a")
	ct.Logf(1, "b")
	ct.Logf(1, "c")
	if got, want := sanitizeString(ct.String()), `  a
  b
  c
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeDump(ct.Dump()), `
  a
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestCallTraceSimple calltrace_test.go:75

  b
    cloudeng.io/debug/goroutine_test.TestCallTraceSimple calltrace_test.go:76

  c
    cloudeng.io/debug/goroutine_test.TestCallTraceSimple calltrace_test.go:77
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCallTraceGoroutines(t *testing.T) {
	ct := &goroutine.CallTrace{}
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
  goroutine L1 launch
    inside L1 goroutine
    inside L1 goroutine
    goroutine L2 launch
      inside L2 goroutine
      inside L2 goroutine
  goroutine L1 launch
    inside L1 goroutine
    inside L1 goroutine
    goroutine L2 launch
      inside L2 goroutine
      inside L2 goroutine
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sanitizeDump(ct.Dump()), `
  a
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines calltrace_test.go:101

  goroutine L1 launch
    cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines calltrace_test.go:107

    inside L1 goroutine
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:109

    inside L1 goroutine
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:111

    goroutine L2 launch
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:112

      inside L2 goroutine
        cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:115

      inside L2 goroutine

  goroutine L1 launch
    testing.tRunner testing.go:991
    cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines calltrace_test.go:107

    inside L1 goroutine
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:109

    inside L1 goroutine
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:111

    goroutine L2 launch
      cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1 calltrace_test.go:112

      inside L2 goroutine
        cloudeng.io/debug/goroutine_test.TestCallTraceGoroutines.func1.1 calltrace_test.go:115

      inside L2 goroutine
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
