// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"

	"cloudeng.io/cmdutil/cmdexec"
)

type logger struct {
	buf *bytes.Buffer
}

func (l *logger) Logf(format string, args ...interface{}) (int, error) {
	return l.buf.WriteString(fmt.Sprintf(format, args...))
}

func newLogger() *logger {
	return &logger{buf: &bytes.Buffer{}}
}

type Variables struct {
	A string
	B int
}

func expand(s string) string {
	switch s {
	case "MINE":
		return "YOURS"
	}
	return os.Getenv(s)
}

func TestExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	l := newLogger()

	output := &bytes.Buffer{}

	opts := []cmdexec.Option{
		cmdexec.WithWorkingDir(tmpDir),
		cmdexec.WithVerbose(true),
		cmdexec.WithLogger(l.Logf),
		cmdexec.WithExpandMapping(expand),
		cmdexec.WithStdout(output),
		cmdexec.WithCommandsPrefix("echo", "hello"),
		cmdexec.WithTemplateVars(Variables{A: "A", B: 42}),
		cmdexec.WithTemplateFuncs(template.FuncMap{
			"add": func(a, b int) int { return a + b },
		})}

	cmds := []string{"world", "{{.A}}", "{{.B}}", "{{add .B 1}}", "${MINE} ${HOME}"}
	home := os.Getenv("HOME")

	dryrun := append([]cmdexec.Option{cmdexec.WithDryRun(true)}, opts...)
	if err := cmdexec.New("test", dryrun...).Run(ctx, cmds...); err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf("[%v]: echo hello world A 42 43 YOURS %v\n", tmpDir, home)
	if got, want := l.buf.String(), expected; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := output.String(), ""; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	l.buf.Reset()
	nodryrun := append([]cmdexec.Option{cmdexec.WithDryRun(false)}, opts...)
	if err := cmdexec.New("test", nodryrun...).Run(ctx, cmds...); err != nil {
		t.Fatal(err)
	}
	if got, want := l.buf.String(), expected; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strings.TrimSpace(output.String()), "hello world A 42 43 YOURS "+home; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
