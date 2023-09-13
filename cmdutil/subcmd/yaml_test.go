// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
)

const (
	toplevel = `name: toplevel
summary: overall documentation
  for toplevel
`

	oneLevel = `name: l0
summary: describe l0
commands:
  - name: l0.1
    summary: l0.1 documentation for l0.1
    arguments:
      - <arg1>
      - <arg2>
  - name: l0.2
    summary: l0.2 documentation for l0.2
    arguments:
      - <arg1>
      - ...
  - name: l0.3
    summary: l0.3 documentation for l0.3
    arguments:
      - ...
  - name: l0.4
    summary: l0.4 documentation for l0.4
    arguments:
      - "[optional-single-argument]"
  - name: l0.5
    summary: l0.5 documentation for l0.5
    # no arguments allowed.
`
)

type exampleFlags struct {
	Flag1 int `subcmd:"flag1,12,flag1"`
}

type runner struct {
	name string
	out  *strings.Builder
}

func (r *runner) cmd(_ context.Context, values interface{}, args []string) error {
	fmt.Fprintf(r.out, "%v: flag: %v, args: %v\n", r.name, values.(*exampleFlags).Flag1, args)
	return nil
}

func TestYAMLCommands(t *testing.T) {
	ctx := context.Background()
	fromYaml := func(spec string) *subcmd.CommandSetYAML {
		csy, err := subcmd.FromYAML([]byte(spec))
		if err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line: %v: %v", line, err)
		}
		return csy
	}

	dispatch := func(cs *subcmd.CommandSetYAML, args ...string) {
		if err := cs.DispatchWithArgs(ctx, os.Args[0], args...); err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line: %v: %v", line, err)
		}
	}

	out := &strings.Builder{}

	assertUsage := func(cs *subcmd.CommandSetYAML, name, usage string) {
		if got, want := cs.Usage(name), usage; !strings.Contains(got, want) {
			_, _, line, _ := runtime.Caller(1)
			t.Errorf("line: %v, got %v does not contain: %v", line, got, want)
		}
	}

	assertRunner := func(cs *subcmd.CommandSetYAML, output string) {
		if got, want := out.String(), output; got != want {
			_, _, line, _ := runtime.Caller(1)

			t.Errorf("line: %v, got %v, want %v", line, got, want)
		}
	}

	tl := &runner{name: "toplevel-example", out: out}

	cs := fromYaml(toplevel)
	cs.Set("toplevel").MustRunnerAndFlags(tl.cmd,
		subcmd.MustRegisteredFlagSet(&exampleFlags{}))
	assertUsage(cs, "toplevel", "overall documentation for toplevel")
	dispatch(cs)
	assertRunner(cs, "toplevel-example: flag: 12, args: []\n")

	cs = fromYaml(oneLevel)
	for _, cmd := range []string{"l0.1", "l0.2", "l0.3", "l0.4", "l0.5"} {
		r := &runner{name: cmd, out: out}
		cs.Set(cmd).MustRunnerAndFlags(r.cmd,
			subcmd.MustRegisteredFlagSet(&exampleFlags{}))
	}

	assertUsage(cs, "l0", "describe l0")

	out.Reset()
	dispatch(cs, "l0.1", "one", "two")
	assertRunner(cs, "l0.1: flag: 12, args: [one two]\n")

}

func ExampleCommandSetYAML_toplevel() {
	cmdSet := subcmd.MustFromYAML(`name: toplevel
summary: overall documentation
  for toplevel
arguments:
  - "[arg - optional]"
`)

	out := &strings.Builder{}

	cmdSet.Set("toplevel").MustRunnerAndFlags(
		(&runner{name: "toplevel", out: out}).cmd,
		subcmd.MustRegisteredFlagSet(&exampleFlags{}))
	if err := cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "-flag1=32", "single-arg"); err != nil {
		panic(err)
	}
	fmt.Println(out.String())
	// Output:
	// toplevel: flag: 32, args: [single-arg]
}

func ExampleCommandSetYAML_multiple() {
	cmdSet := subcmd.MustFromYAML(`name: l0
summary: documentation for l0
commands:
  - name: l0.1
    summary: summary of l0.1
    arguments: # l0.1 expects exactly two arguments.
      - <arg1>
      - <arg2>
  - name: l0.2
    summary: l0.2 summary of l0.2
  - name: l1
    summary: summary of l1
    commands:
      - name: l1.1
        summary: describe l1.1
      - name: l1.2
        summary: describe l1.2
  - name: l2
    commands:
      - name: l2.1
        commands:
          - name: l2.1.1
`)

	out := &strings.Builder{}
	cmdSet.Set("l0.1").MustRunner(
		(&runner{name: "l0.1", out: out}).cmd,
		&exampleFlags{})

	cmdSet.Set("l0.2").MustRunner(
		(&runner{name: "l0.2", out: out}).cmd,
		&exampleFlags{})

	cmdSet.Set("l1", "l1.1").MustRunner(
		(&runner{name: "l1.2", out: out}).cmd,
		&exampleFlags{})

	cmdSet.Set("l1", "l1.2").MustRunner(
		(&runner{name: "l1.2", out: out}).cmd,
		&exampleFlags{})

	cmdSet.Set("l2", "l2.1", "l2.1.1").MustRunner(
		(&runner{name: "l1.2", out: out}).cmd,
		&exampleFlags{})

	if err := cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "l0.1", "-flag1=3", "first-arg", "second-arg"); err != nil {
		panic(err)
	}

	if err := cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "l1", "l1.2", "-flag1=6"); err != nil {
		panic(err)
	}

	fmt.Println(out.String())
	// Output:
	// l0.1: flag: 3, args: [first-arg second-arg]
	// l1.2: flag: 6, args: []
}

func TestYAMLArgumentParsing(t *testing.T) {
	cmdSet := subcmd.MustFromYAML(`name: test
commands:
  - name: c1 # no arguments
  - name: c2
    arguments: # exactly 1 args
      - <arg1>
  - name: c3
    arguments: # exactly 2 args
      - <arg1>
      - <arg2>
  - name: c4
    arguments: # zero or 1.
      - "[optional]"
  - name: c5
    arguments: # at least zero
      - ...
  - name: c6
    arguments: # at least two
      - <arg1>
      - <arg2>
      - ...
`)

	out := &strings.Builder{}
	for _, name := range []string{"c1", "c2", "c3", "c4", "c5", "c6"} {
		cmdSet.Set(name).MustRunner(
			(&runner{name: name, out: out}).cmd,
			&exampleFlags{})
	}

	var err error

	assertError := func(msg string) {
		if err == nil || err.Error() != msg {
			_, _, line, _ := runtime.Caller(1)
			t.Errorf("line: %v, missing or unexpected error: got %v, want %v", line, err, msg)
		}
	}

	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c1", "first-arg")
	assertError("c1: does not accept any arguments")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c2")
	assertError("c2: accepts exactly 1 argument")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c2", "1", "2")
	assertError("c2: accepts exactly 1 argument")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c3", "2")
	assertError("c3: accepts exactly 2 arguments")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c4", "1", "2")
	assertError("c4: accepts at most one argument")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c5")
	if err != nil {
		t.Fatal(err)
	}
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c5", "1")
	if err != nil {
		t.Fatal(err)
	}
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c5", "2")
	if err != nil {
		t.Fatal(err)
	}
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c6", "1")
	assertError("c6: accepts at least 2 arguments")
}

func gorun(t *testing.T, file string, args []string) string {
	allargs := append([]string{"run", filepath.Join("testdata", file)}, args...)
	cmd := exec.Command("go", allargs...)
	t.Log(strings.Join(cmd.Args, " "))
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func TestYAMLCompatibility(t *testing.T) {
	for _, tc := range []string{"toplevel", "onelevel", "multilevel"} {
		for _, args := range [][]string{
			{},
			{"l0.1"},
		} {
			want := gorun(t, tc+".go", args)
			got := gorun(t, tc+"_yaml.go", args)
			got = strings.Replace(got, tc+"_yaml", tc, 1)
			fmt.Printf("..%s\n", want)
			fmt.Printf("..%s\n", got)
			if got != want {
				t.Errorf("%v: %v: got %v, want %v", tc, args, got, want)
			}
		}
	}
}

func TestFErrors(t *testing.T) {
	r := &runner{name: "a", out: nil}
	cmdSet := subcmd.MustFromYAMLTemplate(oneLevel)
	var notastruct int
	err := cmdSet.Set("l0.1").Runner(r.cmd, &notastruct)
	if err == nil || err.Error() != "*int is not a pointer to a struct" {
		t.Errorf("missing or wrong error: %v", err)
	}
}
