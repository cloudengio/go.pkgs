// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pproftrace_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"cloudeng.io/debug/goroutines/pproftrace"
)

var spawned = make(chan struct{})

func runner(ctx context.Context, ch, dch chan struct{}) {
	go func() {
		close(spawned)
		time.Sleep(time.Second)
		<-ch
		close(dch)
	}()
}

func TestRunAndFormat(t *testing.T) {
	ctx := context.Background()
	key, value := "testing", t.Name()
	ch := make(chan struct{})
	dch := make(chan struct{})

	pproftrace.Run(ctx, key, value, func(ctx context.Context) {
		runner(ctx, ch, dch)
	})
	<-spawned

	exists, err := pproftrace.LabelExists(key, value)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := exists, true; got != want {
		output, _ := pproftrace.Format(key, value)
		t.Errorf("got %v, want %v", got, want)
		t.Logf("error: %v %v does not exist in: %q", key, value, output)
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
}
