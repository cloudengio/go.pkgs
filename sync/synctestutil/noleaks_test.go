// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package synctestutil_test

import (
	"testing"
	"time"

	"cloudeng.io/sync/synctestutil"
)

type fakeErrorf struct {
	calls int
	extra int
}

func (f *fakeErrorf) Errorf(format string, args ...interface{}) {
	f.calls++
	f.extra = args[0].(int)
}

func TestNoLeaks(t *testing.T) {
	er := &fakeErrorf{}

	// Simple case with no goroutines.
	fn := synctestutil.AssertNoGoroutines(er)
	fn()
	if got, want := er.calls, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
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
