// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package shells_test

import (
	"bytes"
	"context"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/cmdutil/subcmd/shells"
)

type globalFlags struct {
	Config  string `subcmd:"config,,config file [path]: optional"`
	Verbose bool   `subcmd:"verbose,false,enable verbose output"`
}

type listFlags struct {
	Limit int  `subcmd:"limit,10,max items"`
	All   bool `subcmd:"all,false,include all"`
}

type getFlags struct {
	Format string `subcmd:"format,json,output format: json or yaml"`
}

func noopRunner(_ context.Context, _ any, _ []string) error {
	return nil
}

func newTestYAMLCLI(out *bytes.Buffer) *subcmd.CommandSetYAML {
	const spec = `name: mytool
summary: test cli
commands:
  - name: list
    summary: list items
  - name: db
    summary: database commands
    commands:
      - name: get
        summary: 'get a record: by id'
        arguments:
          - <id> - record id
      - name: reset
        summary: reset database
  - name: completions
    summary: generate shell completions
    arguments:
      - <shell> - shell name (bash or zsh)
`
	cmdSet := subcmd.MustFromYAML(spec)
	var gf globalFlags
	cmdSet.WithGlobalFlags(subcmd.GlobalFlagSet().MustRegisterFlagStruct(&gf, nil, nil))
	cmdSet.Set("list").MustRunner(noopRunner, &listFlags{})
	cmdSet.Set("db", "get").MustRunner(noopRunner, &getFlags{})
	cmdSet.Set("db", "reset").MustRunner(noopRunner, &struct{}{})
	cmdSet.Set("completions").MustRunner(shells.Runner(cmdSet.CommandSet, out), &shells.Flags{})
	return cmdSet
}

func TestGenerateErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := shells.Generate(&buf, "bash", "mytool", nil); err == nil {
		t.Fatal("expected error for nil command set")
	}
	cmdSet := newTestYAMLCLI(&buf)
	if err := shells.Generate(&buf, "fish", "mytool", cmdSet.CommandSet); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}

func TestRunnerAndDispatch(t *testing.T) {
	for _, sh := range shells.SupportedShells() {
		t.Run(sh, func(t *testing.T) {
			var out bytes.Buffer
			cmdSet := newTestYAMLCLI(&out)
			err := cmdSet.DispatchWithArgs(t.Context(), "mytool", "completions", "--program=my-tool", sh)
			if err != nil {
				t.Fatalf("DispatchWithArgs failed: %v", err)
			}
			got := out.String()
			if !strings.Contains(got, "my-tool") {
				t.Errorf("expected output to reference program my-tool, got:\n%s", got)
			}
			if sh == shells.BashShell {
				checkBashSyntax(t, got)
			} else {
				checkZshSyntax(t, got)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	var out bytes.Buffer
	fs := subcmd.MustRegisteredFlagSet(&listFlags{})
	listCmd := subcmd.NewCommand("list", fs, noopRunner, subcmd.WithoutArguments())
	listCmd.Document("list items")

	// Construct CommandSet first so completions command can reference it.
	var cmdSet *subcmd.CommandSet
	compCmd := shells.NewCommand("completions", subcmd.NewCommandSet(listCmd), &out)
	cmdSet = subcmd.NewCommandSet(listCmd, compCmd)
	_ = cmdSet

	if err := cmdSet.DispatchWithArgs(t.Context(), "mytool", "completions", "bash"); err != nil {
		t.Fatalf("DispatchWithArgs failed: %v", err)
	}
	if !strings.Contains(out.String(), "_mytool_completions") {
		t.Logf("generated bash script:\n%s", out.String())
	}
	checkBashSyntax(t, out.String())
}

func TestTopLevelCommand(t *testing.T) {
	var out bytes.Buffer
	fs := subcmd.MustRegisteredFlagSet(&listFlags{})
	tl := subcmd.NewCommand("single", fs, noopRunner, subcmd.WithoutArguments())
	tl.Document("single command tool")
	cmdSet := subcmd.NewCommandSet()
	cmdSet.TopLevel(tl)

	if err := shells.Bash(&out, "single", cmdSet); err != nil {
		t.Fatalf("Bash failed: %v", err)
	}
	bashScript := out.String()
	checkBashSyntax(t, bashScript)
	if !strings.Contains(bashScript, "--limit") || !strings.Contains(bashScript, "--all") {
		t.Errorf("expected top-level flags in bash script:\n%s", bashScript)
	}

	out.Reset()
	if err := shells.Zsh(&out, "single", cmdSet); err != nil {
		t.Fatalf("Zsh failed: %v", err)
	}
	zshScript := out.String()
	checkZshSyntax(t, zshScript)
	if !strings.Contains(zshScript, "--limit") || !strings.Contains(zshScript, "--all") {
		t.Errorf("expected top-level flags in zsh script:\n%s", zshScript)
	}
}

func TestBashCompletionsRuntime(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}
	var out bytes.Buffer
	cmdSet := newTestYAMLCLI(&out)
	if err := shells.Bash(&out, "my-tool", cmdSet.CommandSet); err != nil {
		t.Fatalf("Bash failed: %v", err)
	}
	script := out.String()
	checkBashSyntax(t, script)

	cases := []struct {
		name   string
		words  string
		cword  int
		want   []string
		absent []string
	}{
		{
			name:  "top-level subcommands",
			words: `("my-tool" "")`,
			cword: 1,
			want:  []string{"list", "db", "completions", "help", "vcsinfo"},
		},
		{
			name:  "global flags",
			words: `("my-tool" "--")`,
			cword: 1,
			want:  []string{"--config", "--verbose", "--help"},
		},
		{
			name:   "nested subcommands",
			words:  `("my-tool" "db" "")`,
			cword:  2,
			want:   []string{"get", "reset"},
			absent: []string{"list"},
		},
		{
			name:   "nested leaf flags",
			words:  `("my-tool" "db" "get" "--")`,
			cword:  3,
			want:   []string{"--format", "--help"},
			absent: []string{"--limit"},
		},
		{
			name:  "help subcommand targets",
			words: `("my-tool" "help" "")`,
			cword: 2,
			want:  []string{"list", "db", "completions"},
		},
		{
			name:  "value-taking flag consumes next word",
			words: `("my-tool" "--config" "my.yaml" "")`,
			cword: 3,
			want:  []string{"list", "db"},
		},
		{
			name:   "value-taking flag waiting for value",
			words:  `("my-tool" "--config" "")`,
			cword:  2,
			absent: []string{"list", "db"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snippet := script + "\n" +
				"COMP_WORDS=" + tc.words + "\n" +
				"COMP_CWORD=" + strconv.Itoa(tc.cword) + "\n" +
				"_my_tool_completions\n" +
				"printf '%s\\n' \"${COMPREPLY[@]}\"\n"
			cmd := exec.CommandContext(t.Context(), "bash", "-c", snippet)
			gotBytes, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bash execution failed: %v\noutput:\n%s", err, string(gotBytes))
			}
			got := strings.Fields(string(gotBytes))
			for _, w := range tc.want {
				if !slices.Contains(got, w) {
					t.Errorf("missing expected completion %q in %v", w, got)
				}
			}
			for _, a := range tc.absent {
				if slices.Contains(got, a) {
					t.Errorf("unexpected completion %q in %v", a, got)
				}
			}
		})
	}
}

func TestZshScriptContent(t *testing.T) {
	var out bytes.Buffer
	cmdSet := newTestYAMLCLI(&out)
	if err := shells.Zsh(&out, "my-tool", cmdSet.CommandSet); err != nil {
		t.Fatalf("Zsh failed: %v", err)
	}
	script := out.String()
	checkZshSyntax(t, script)

	for _, want := range []string{
		"#compdef my-tool",
		"_my_tool()",
		"_my_tool_db()",
		"_my_tool_db_get()",
		`'get:get a record\: by id'`,
		`config file \[path\]\: optional`,
		`compdef _my_tool 'my-tool'`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("expected %q in zsh script:\n%s", want, script)
		}
	}
}

func checkBashSyntax(t *testing.T, script string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		return
	}
	cmd := exec.CommandContext(t.Context(), "bash", "-n")
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s\nscript:\n%s", err, string(out), script)
	}
}

func checkZshSyntax(t *testing.T, script string) {
	t.Helper()
	if _, err := exec.LookPath("zsh"); err != nil {
		return
	}
	cmd := exec.CommandContext(t.Context(), "zsh", "-n")
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zsh -n failed: %v\n%s\nscript:\n%s", err, string(out), script)
	}
}
