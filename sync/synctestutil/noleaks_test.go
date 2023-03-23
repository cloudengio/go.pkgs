// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package synctestutil_test

import (
	"testing"
	"time"

	"cloudeng.io/debug/goroutines"
	"cloudeng.io/sync/synctestutil"
)

type fakeErrorf struct {
	calls int
}

func (f *fakeErrorf) Errorf(_ string, _ ...interface{}) {
	f.calls++
}

func mustGet(t *testing.T) []*goroutines.Goroutine {
	gs, err := goroutines.Get()
	if err != nil {
		t.Fatal(err)
	}
	return gs
}

func TestNoLeaks(t *testing.T) {
	er := &fakeErrorf{}

	// Simple case with no goroutines.
	before := mustGet(t)
	fn := synctestutil.AssertNoGoroutines(er)
	fn()
	after := mustGet(t)
	if got, want := er.calls, 0; got != want {
		t.Logf("before: %v: %v", len(before), goroutines.Format(before...))
		t.Logf("after: %v: %v", len(after), goroutines.Format(after...))
		t.Fatalf("got %v, want %v", got, want)
	}

	stop := make(chan struct{})
	running := make(chan struct{})
	fn = synctestutil.AssertNoGoroutines(er)
	go func() {
		close(running)
		<-stop
	}()
	<-running
	fn()
	if got, want := er.calls, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	close(stop)

	er = &fakeErrorf{}
	fn = synctestutil.AssertNoGoroutinesRacy(er, time.Minute)
	go func() {
		time.Sleep(time.Millisecond * 100)
	}()
	fn()
	if got, want := er.calls, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
