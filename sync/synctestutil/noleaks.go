// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package synctestutil

import (
	"time"

	"cloudeng.io/debug/goroutines"
)

// Errorf is called when an error is encountered and is defined so that
// testing.T and testing.B implement Errorf.
type Errorf interface {
	Errorf(format string, args ...interface{})
}

// AssertNoGoroutines is used to detect goroutine leaks.
//
// Usage is as shown below:
//
//	func TestExample(t *testing.T) {
//		defer synctestutil.AssertNoGoroutines(t, time.Second)()
//		...
//	}
//
// Note that in the example above AssertNoGoroutines returns a function
// that is immediately defered. The call to AssertNoGoroutines records
// the currently goroutines and the returned function will compare that
// initial set to those running when it is invoked. Hence, the above
// example is equivalent to:
//
//	func TestExample(t *testing.T) {
//		fn := synctestutil.AssertNoGoroutines(t, time.Second)
//		...
//		fn()
//	}
func AssertNoGoroutines(t Errorf) func() {
	bycreator, err := getGoroutines()
	if err != nil {
		t.Errorf("NoLeaks: failed to parse goroutine output %v", err)
		return func() {}
	}
	return func() {
		cbycreator, err := getGoroutines()
		if err != nil {
			t.Errorf("NoLeaks: failed to parse goroutine output %v", err)
			return
		}
		left := compare(bycreator, cbycreator)
		if len(left) != 0 {
			t.Errorf("%d extra Goroutines outstanding:\n %s",
				len(left), goroutines.Format(left...))
		}

	}
}

// AssertNoGoroutinesRacy is like AssertNoGoroutines but allows for
// a grace period for goroutines to terminate.
func AssertNoGoroutinesRacy(t Errorf, wait time.Duration) func() {
	bycreator, err := getGoroutines()
	if err != nil {
		t.Errorf("NoLeaks: failed to parse goroutine output %v", err)
		return func() {}
	}
	return func() {
		backoff := 100 * time.Millisecond
		start := time.Now()
		until := start.Add(wait)
		for {
			cbycreator, err := getGoroutines()
			if err != nil {
				t.Errorf("NoLeaks: failed to parse goroutine output %v", err)
				return
			}
			left := compare(bycreator, cbycreator)
			if len(left) == 0 {
				return
			}
			if time.Now().After(until) {
				t.Errorf("%d extra Goroutines outstanding after %v:\n %s",
					len(left), wait, goroutines.Format(left...))
				return
			}
			time.Sleep(backoff)
			if backoff *= 2; backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}

func getGoroutines() (map[string]*goroutines.Goroutine, error) {
	gs, err := goroutines.Get()
	if err != nil {
		return nil, err
	}
	bycreator := map[string]*goroutines.Goroutine{}
	for _, g := range gs {
		key := ""
		if g.Creator != nil {
			key = g.Creator.Call
		}
		bycreator[key] = g
	}
	return bycreator, nil
}

func compare(before, after map[string]*goroutines.Goroutine) []*goroutines.Goroutine {
	var left []*goroutines.Goroutine
	for k, g := range after {
		if _, ok := before[k]; !ok {
			left = append(left, g)
		}
	}
	return left
}
