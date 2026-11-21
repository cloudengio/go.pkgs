// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

var trickyArgs = []string{
	"", "plain", "with space", "it's", "'", "''", `"double"`, "$HOME", "`id`",
	`back\slash`, "a;b|c&d", "*?[x]", "new\nline", "~", "!x", "tab\there",
}

func TestShellQuote(t *testing.T) {
	for _, arg := range trickyArgs {
		out, err := exec.Command("/bin/sh", "-c", "printf %s "+ShellQuote(arg)).Output() //nolint:gosec
		if err != nil {
			t.Fatalf("%q: %v", arg, err)
		}
		if got := string(out); got != arg {
			t.Errorf("%q: shell produced %q", arg, got)
		}
	}
}

func TestShellJoin(t *testing.T) {
	if got := ShellJoin(); got != "" {
		t.Errorf("got %q, want an empty command", got)
	}
	out, err := exec.Command("/bin/sh", "-c", `printf '%s|' `+ShellJoin(trickyArgs...)).Output() //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	for _, a := range trickyArgs {
		want.WriteString(a)
		want.WriteString("|")
	}
	if got := string(out); got != want.String() {
		t.Errorf("got %q, want %q", got, want.String())
	}
}

// fakeLoginShell sets SHELL to a script that reports its arguments, for the
// server to find when running a command.
func fakeLoginShell(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shell")
	script := "#!/bin/sh\nprintf 'login-shell %s|%s|%s\\n' \"$1\" \"$2\" \"$3\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	t.Setenv("SHELL", path)
}

func TestRunLoginShell(t *testing.T) {
	fakeLoginShell(t)
	c, _, _ := runFixture(t)
	ctx := timeoutCtx(t)
	cmd := "echo 'a b'; echo $HOME"

	var stdout bytes.Buffer
	if err := c.Run(ctx, cmd, WithLoginShell(), WithStdout(&stdout)); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "login-shell -l|-c|"+cmd+"\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Without the option the command is passed on untouched.
	stdout.Reset()
	if err := c.Run(ctx, "echo plain", WithStdout(&stdout)); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "plain\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunLoginShellDefault(t *testing.T) {
	t.Setenv("SHELL", "")
	c, _, _ := runFixture(t)
	var stdout bytes.Buffer
	err := c.Run(timeoutCtx(t), "echo ok; exit 4", WithLoginShell(), WithStdout(&stdout))
	var exitErr *ssh.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitStatus() != 4 {
		t.Fatalf("got %v, want an exit status of 4", err)
	}
	if got, want := stdout.String(), "ok\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
