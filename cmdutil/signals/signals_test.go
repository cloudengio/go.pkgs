// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package signals_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/cmdutil/signals"
)

func runSubprocess(t *testing.T, args []string) (*exec.Cmd, io.Reader) {
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cl := []string{"run", filepath.Join("testdata", "signal_main.go")}
	cl = append(cl, args...)
	cmd := exec.Command("go", cl...)
	cmd.Stdout = wr
	cmd.Stderr = wr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
	}
	return cmd, rd
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, handler := signals.NotifyWithCancel(ctx, os.Interrupt)

	go func() {
		cancel()
	}()

	sig := handler.WaitForSignal()
	if got, want := sig.String(), "context canceled"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMultipleCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, handler := signals.NotifyWithCancel(ctx, os.Interrupt)
	out := []string{}
	mu := sync.Mutex{}
	writeString := func(m string) {
		mu.Lock()
		defer mu.Unlock()
		out = append(out, m)
	}
	getString := func() string {
		mu.Lock()
		defer mu.Unlock()
		return strings.Join(out, "..")
	}
	handler.RegisterCancel(
		func() {
			writeString("a")
		},
		func() {
			writeString("b")
		},
	)

	go func() {
		cancel()
	}()

	sig := handler.WaitForSignal()
	if got, want := sig.String(), "context canceled"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	time.Sleep(time.Second)
	if got, want := getString(), "a..b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
