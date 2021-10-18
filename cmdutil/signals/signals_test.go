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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/cmdutil/expect"
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

func TestSignal(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	runCmd := func(args ...string) (*exec.Cmd, int, *expect.Lines) {
		cmd, rd := runSubprocess(t, args)
		st := expect.NewLineStream(rd)
		if err := st.ExpectEventuallyRE(ctx, regexp.MustCompile(`PID=\d+`)); err != nil {
			t.Fatal(err)
		}
		_, line := st.LastMatch()
		pid, err := strconv.ParseInt(line[strings.Index(line, "=")+1:], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return cmd, int(pid), st
	}

	var wg sync.WaitGroup
	wg.Add(1)
	cmd, _, st := runCmd("--debounce=5s")
	go func() {
		// Make sure that multiple signals in quick succession do not
		// cause the process to exit.
		cmd.Process.Signal(syscall.SIGINT)
		cmd.Process.Signal(syscall.SIGINT)
		cmd.Process.Signal(syscall.SIGINT)
		wg.Done()
	}()

	if err := st.ExpectEventuallyRE(ctx, regexp.MustCompile(`CANCEL PID=\d+`)); err != nil {
		t.Fatal(err)
	}

	if err := st.ExpectNext(ctx, "interrupt"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	wg.Wait()

	// Make sure that a second signal after the debounce period leads to
	// an exit.
	cmd, _, st = runCmd("--debounce=250ms")
	go func() {
		cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(time.Millisecond * 250)
		cmd.Process.Signal(syscall.SIGINT)
	}()
	if err := st.ExpectEventuallyRE(ctx, regexp.MustCompile(`CANCEL PID=\d+`)); err != nil {
		t.Fatal(err)
	}
	if err := st.ExpectNext(ctx, "exit status 1"); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	if err == nil || err.Error() != "exit status 1" {
		t.Errorf("unexpected error: %v", err)
	}

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
