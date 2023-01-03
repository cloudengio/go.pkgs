// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/debug/goroutines"
	"cloudeng.io/path/gopkgpath"
)

func wrappedWaitForIt(wg *sync.WaitGroup, wait chan struct{}, n int64) {
	if n == 0 {
		waitForIt(wg, wait)
	} else {
		wrappedWaitForIt(wg, wait, n-1)
	}
}

func waitForIt(wg *sync.WaitGroup, wait chan struct{}) {
	wg.Done()
	<-wait
}

func runGoA(wg *sync.WaitGroup, wait chan struct{}) {
	go waitForIt(wg, wait)
}

func runGoB(wg *sync.WaitGroup, wait chan struct{}) {
	go wrappedWaitForIt(wg, wait, 3)
}

func runGoC(wg *sync.WaitGroup, wait chan struct{}) {
	go func() {
		wg.Done()
		<-wait
	}()
}

func TestGet(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	wait := make(chan struct{})
	runGoA(&wg, wait)
	runGoB(&wg, wait)
	runGoC(&wg, wait)
	wg.Wait()
	gs, err := goroutines.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(gs) < 4 {
		t.Errorf("Got %d goroutines, expected at least 4", len(gs))
	}
	bycreator := map[string]*goroutines.Goroutine{}
	for _, g := range gs {
		key := ""
		if g.Creator != nil {
			key = g.Creator.Call
		}
		bycreator[key] = g
	}

	pkgPath, _ := gopkgpath.Caller()
	pkgPath += "_test."
	a := bycreator[pkgPath+"runGoA"]
	switch {
	case a == nil:
		for _, g := range gs {
			if g.Creator != nil {
				t.Logf("%v", g.Creator.Call)
			}
		}
		fmt.Printf("><>< %v\n", bycreator)
		panic("runGoA is missing")
	case len(a.Stack) < 1:
		t.Errorf("got %d expected at least 1: %s", len(a.Stack), goroutines.Format(a))
	case !strings.HasPrefix(a.Stack[0].Call, pkgPath+"waitForIt"):
		t.Errorf("got %s, wanted it to start with %swaitForIt",
			a.Stack[0].Call, pkgPath)
	}
	b := bycreator[pkgPath+"runGoB"]
	if b == nil {
		t.Errorf("runGoB is missing")
	} else if len(b.Stack) < 5 {
		t.Errorf("got %d expected at least 5: %s", len(b.Stack), goroutines.Format(b))
	}
	c := bycreator[pkgPath+"runGoC"]
	if c == nil {
		t.Errorf("runGoC is missing")
	} else if len(c.Stack) < 1 {
		t.Errorf("got %d expected at least 1: %s", len(c.Stack), goroutines.Format(c))
	}
	// Allow goroutines to exit.
	close(wait)
}

func TestGetIgnore(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	wait := make(chan struct{})
	runGoA(&wg, wait)
	runGoB(&wg, wait)
	runGoC(&wg, wait)
	wg.Wait()
	gs, err := goroutines.Get("runGoA", "runGoB")
	if err != nil {
		t.Fatal(err)
	}
	bycreator := map[string]*goroutines.Goroutine{}
	for _, g := range gs {
		key := ""
		if g.Creator != nil {
			key = g.Creator.Call
		}
		bycreator[key] = g
	}

	pkgPath, _ := gopkgpath.Caller()
	pkgPath += "_test."
	for _, ignored := range []string{"runGoA", "runGoB"} {
		if _, ok := bycreator[pkgPath+ignored]; ok {
			t.Errorf("%v should have been recorded", ignored)
		}
	}
	if _, ok := bycreator[pkgPath+"runGoC"]; !ok {
		t.Errorf("%v should have been recorded", "runGoC")
	}
}

func TestFormat(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	wait := make(chan struct{})
	runGoA(&wg, wait)
	runGoB(&wg, wait)
	runGoC(&wg, wait)
	wg.Wait()

	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	close(wait)

	gs, err := goroutines.Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	if formatted := goroutines.Format(gs...); string(buf) != formatted {
		t.Errorf("got:\n%s\nwanted:\n%s\n", formatted, buf)
	}
}
