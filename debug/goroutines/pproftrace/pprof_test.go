// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pproftrace_test

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cloudeng.io/debug/goroutines/pproftrace"
)

var spawned chan struct{}

func runner(delay time.Duration, ch, dch chan struct{}) {
	go func() {
		for i := 1; i < 10000; i++ {
			_ = rand.Int63n(int64(i))
		}
		close(spawned)
		time.Sleep(delay)
		<-ch
		close(dch)
	}()
}

func testRunAndFormat(t *testing.T, delay time.Duration) error {
	ctx := context.Background()
	key, value := "testing", t.Name()
	ch := make(chan struct{})
	dch := make(chan struct{})

	pproftrace.Run(ctx, key, value, func(ctx context.Context) {
		runner(delay, ch, dch)
	})
	<-spawned

	exists, err := pproftrace.LabelExists(key, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := exists, true; got != want {
		output, _ := pproftrace.Format(key, value)
		t.Logf("error: %v %v does not exist in: %q", key, value, output)
		return fmt.Errorf("got %v, want %v", got, want)
	}
	output, err := pproftrace.Format(key, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output, "pproftrace_test.runner.func1"; !strings.Contains(got, want) {
		t.Errorf("got %v does not contain %v", got, want)
	}

	close(ch)
	<-dch

	exists, err = pproftrace.LabelExists(key, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := exists, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	output, err = pproftrace.Format(key, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output, "pproftrace_test.runner.func1"; strings.Contains(got, want) {
		t.Errorf("got %v contains %v", got, want)
	}
	return nil
}

func TestRunAndFormat(t *testing.T) {
	var err error
	delay := time.Second
	for i := 0; i < 3; i++ {
		spawned = make(chan struct{})
		err = testRunAndFormat(t, delay)
		if err == nil {
			return
		}
		delay += time.Second
	}
	if err != nil {
		t.Fatal(err)
	}
}
