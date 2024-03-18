// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"cloudeng.io/cmdutil"
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
	V string
}

func expand(s string) string {
	switch s {
	case "MINE":
		return "YOURS"
	case "MY_VAR":
		return "ENV_VAR"
	}
	return os.Getenv(s)
}

func ExampleRunner() {
	ctx := context.Background()
	os.Setenv("ENV_VAR", "ENV_VAR_VAL")
	err := cmdexec.New("test",
		cmdexec.WithTemplateVars(struct{ A string }{A: "value"}),
	).Run(ctx, "echo", "{{.A}}", "$ENV_VAR")
	if err != nil {
		fmt.Println(err)
	}

	// Output:
	// value ENV_VAR_VAL
}

func TestExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	l := newLogger()

	output := &bytes.Buffer{}

	testcmd := []string{
		"go", "run", filepath.Join("testdata", "main.go"),
	}
	os.Setenv("os_env_var", "something")
	env := cmdexec.AppendToOSEnv("os_env_var_child=something_else")
	opts := []cmdexec.Option{
		cmdexec.WithEnv(env),
		cmdexec.WithWorkingDir(tmpDir),
		cmdexec.WithVerbose(true),
		cmdexec.WithLogger(l.Logf),
		cmdexec.WithExpandMapping(expand),
		cmdexec.WithStdout(output),
		cmdexec.WithCommandsPrefix(testcmd...),
		cmdexec.WithTemplateVars(Variables{A: "A", B: 42, V: "${MY_VAR}"}),
		cmdexec.WithTemplateFuncs(template.FuncMap{
			"add": func(a, b int) int { return a + b },
		})}

	cmds := []string{"world", "{{.A}}", "{{.B}}", "{{add .B 1}}", "${MINE} ${os_env_var}", "os_env_var_child", "{{.V}}"}
	envVar := os.Getenv("os_env_var")

	dryrun := append([]cmdexec.Option{cmdexec.WithDryRun(true)}, opts...)
	if err := cmdexec.New("test", dryrun...).Run(ctx, cmds...); err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf(
		"[%v]: %v world A 42 43 YOURS %v os_env_var_child ENV_VAR\n",
		tmpDir, strings.Join(testcmd, " "), envVar)
	if got, want := l.buf.String(), expected; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := output.String(), ""; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if err := cmdutil.CopyAll("testdata", tmpDir, false); err != nil {
		t.Fatal(err)
	}
	l.buf.Reset()
	nodryrun := append([]cmdexec.Option{cmdexec.WithDryRun(false)}, opts...)
	if err := cmdexec.New("test", nodryrun...).Run(ctx, cmds...); err != nil {
		t.Fatal(err)
	}
	if got, want := l.buf.String(), expected; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strings.TrimSpace(output.String()), "world A 42 43 YOURS "+envVar+" something_else ENV_VAR"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
