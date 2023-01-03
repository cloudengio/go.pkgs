// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package signals_test

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/cmdutil/expect"
)

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
	cmd, pid, st := runCmd("--debounce=5s")
	go func() {
		// Make sure that multiple signals in quick succession do not
		// cause the process to exit.
		_ = syscall.Kill(pid, syscall.SIGINT)
		_ = syscall.Kill(pid, syscall.SIGINT)
		_ = syscall.Kill(pid, syscall.SIGINT)
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
	cmd, pid, st = runCmd("--debounce=250ms")
	go func() {
		_ = syscall.Kill(pid, syscall.SIGINT)
		time.Sleep(time.Millisecond * 300)
		_ = syscall.Kill(pid, syscall.SIGINT)
	}()
	if err := st.ExpectEventuallyRE(ctx, regexp.MustCompile(`CANCEL PID=\d+`)); err != nil {
		t.Fatal(err)
	}
	if err := st.ExpectNextRE(ctx, regexp.MustCompile("^exit status 1$")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	if err == nil || err.Error() != "exit status 1" {
		t.Errorf("unexpected error: %v", err)
	}
}
