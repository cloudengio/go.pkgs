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
      - <arg1> - arg 1
      - <arg2> - arg 2
  - name: l0.2
    summary: l0.2 documentation for l0.2
    args:
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

	oneLevelTabs = `name: l0
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

func (r *runner) cmd(_ context.Context, values any, args []string) error {
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

	assertRunner := func(_ *subcmd.CommandSetYAML, output string) {
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

	for _, spec := range []string{oneLevel, subcmd.SanitizeYAML(oneLevelTabs)} {
		cs = fromYaml(spec)
		for _, cmd := range []string{"l0.1", "l0.2", "l0.3", "l0.4", "l0.5"} {
			r := &runner{name: cmd, out: out}
			cs.Set(cmd).MustRunnerAndFlags(r.cmd,
				subcmd.MustRegisteredFlagSet(&exampleFlags{}))
		}
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
    arguments: # zero or 1, ie. at most one.
      - '[optional]'
  - name: c5
    arguments: # at least zero
      - ...
  - name: c6
    arguments: # at least two
      - <arg1>
      - <arg2>
      - ...
  - name: c7
    arguments: # at least two
      - <arg1>
      - <opt>...
  - name: c8
    arguments: # zero or 1, ie. at most one.
      - |
        "[optional]" - with an explanation
  - name: c9
    arguments: # at least zero
      - ... - with an explanation
  - name: c10
    arguments: # at least two
      - <arg1>
      - <arg2>
      - ... - with an explanation
  - name: c11
    arguments: # at least two
      - <arg1>
      - <opt>... - with an explanation
  `)

	out := &strings.Builder{}
	for _, name := range []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10", "c11"} {
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
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c8", "1", "2")
	assertError("c8: accepts at most one argument")

	for _, c := range []string{"c5", "c9"} {
		err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], c)
		if err != nil {
			t.Fatal(err)
		}
		err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], c, "1")
		if err != nil {
			t.Fatal(err)
		}
		err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], c, "1", "2")
		if err != nil {
			t.Fatal(err)
		}
	}
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c6", "1")
	assertError("c6: accepts at least 2 arguments")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c7")
	assertError("c7: accepts at least 1 argument")

	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c10", "1")
	assertError("c10: accepts at least 2 arguments")
	err = cmdSet.DispatchWithArgs(context.Background(), os.Args[0], "c11")
	assertError("c11: accepts at least 1 argument")

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

func TestSetPreHooks(t *testing.T) {
	ctx := context.Background()
	hookCalled := map[string]int{}

	makeHook := func(tag string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			hookCalled[tag]++
			return ctx, nil, nil
		}
	}

	noopRunner := func(_ context.Context, _ any, _ []string) error { return nil }

	const spec = `name: root
summary: root
commands:
  - name: a
    summary: a
  - name: b
    summary: b
    commands:
      - name: b1
        summary: b1
      - name: b2
        summary: b2
`
	setup := func(t *testing.T) *subcmd.CommandSetYAML {
		t.Helper()
		cs := subcmd.MustFromYAML(spec)
		cs.Set("a").MustRunner(noopRunner, &struct{}{})
		cs.Set("b", "b1").MustRunner(noopRunner, &struct{}{})
		cs.Set("b", "b2").MustRunner(noopRunner, &struct{}{})
		return cs
	}

	t.Run("leaf command", func(t *testing.T) {
		cs := setup(t)
		hookCalled = map[string]int{}
		cs.Set("a").MustSetPreHooks(makeHook("h"))
		if err := cs.DispatchWithArgs(ctx, "root", "a"); err != nil {
			t.Fatal(err)
		}
		// no hooks.
		if err := cs.DispatchWithArgs(ctx, "root", "b", "b1"); err != nil {
			t.Fatal(err)
		}
		if len(hookCalled) != 1 {
			t.Errorf("unexpected hook tags called: got %v, want [h]", hookCalled)
		}
		if hookCalled["h"] != 1 {
			t.Errorf("hook called %d times, want 1", hookCalled["h"])
		}
	})

	t.Run("intermediate node applies to all descendants", func(t *testing.T) {
		cs := setup(t)
		hookCalled = map[string]int{}
		cs.Set("b").MustSetPreHooks(makeHook("h"))
		if err := cs.DispatchWithArgs(ctx, "root", "b", "b1"); err != nil {
			t.Fatal(err)
		}
		if err := cs.DispatchWithArgs(ctx, "root", "b", "b2"); err != nil {
			t.Fatal(err)
		}
		if len(hookCalled) != 1 {
			t.Errorf("unexpected hook tags called: got %v, want [h]", hookCalled)
		}
		if hookCalled["h"] != 2 {
			t.Errorf("hook called %d times for b1+b2, want 2", hookCalled["h"])
		}
	})

	t.Run("sibling unaffected", func(t *testing.T) {
		cs := setup(t)
		hookCalled = map[string]int{}
		cs.Set("b").MustSetPreHooks(makeHook("h"))
		if err := cs.DispatchWithArgs(ctx, "root", "a"); err != nil {
			t.Fatal(err)
		}
		if len(hookCalled) != 0 {
			t.Errorf("unexpected hook tags called: got %v, want none", hookCalled)
		}
		if hookCalled["h"] != 0 {
			t.Errorf("hook called %d times for sibling 'a', want 0", hookCalled["h"])
		}
	})

	t.Run("error on unknown command", func(t *testing.T) {
		cs := setup(t)
		err := cs.Set("nonexistent").SetPreHooks(makeHook("h"))
		if err == nil || !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("expected error mentioning nonexistent, got %v", err)
		}
	})
}

func ExampleCurrentCommand_SetPreHooks() {
	ctx := context.Background()
	logged := []string{}

	traceHook := subcmd.PreHook(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		logged = append(logged, "pre")
		post := func(_ context.Context) error {
			logged = append(logged, "post")
			return nil
		}
		return ctx, post, nil
	})

	out := &strings.Builder{}
	cmdSet := subcmd.MustFromYAML(`name: tool
summary: example tool
commands:
  - name: sub1
    summary: first sub
  - name: sub2
    summary: second sub
`)
	cmdSet.Set("sub1").MustRunner((&runner{name: "sub1", out: out}).cmd, &exampleFlags{})
	cmdSet.Set("sub2").MustRunner((&runner{name: "sub2", out: out}).cmd, &exampleFlags{})

	// Apply the tracing hook to sub1 only.
	cmdSet.Set("sub1").MustSetPreHooks(traceHook)

	if err := cmdSet.DispatchWithArgs(ctx, "tool", "sub1"); err != nil {
		panic(err)
	}
	fmt.Println(strings.Join(logged, ","))
	// Output:
	// pre,post
}

func TestAppendPreHooks(t *testing.T) {
	ctx := context.Background()

	const spec = `name: root
summary: root
commands:
  - name: a
    summary: a
  - name: b
    summary: b
    commands:
      - name: b1
        summary: b1
      - name: b2
        summary: b2
`
	noopRunner := func(_ context.Context, _ any, _ []string) error { return nil }

	setup := func(t *testing.T) *subcmd.CommandSetYAML {
		t.Helper()
		cs := subcmd.MustFromYAML(spec)
		cs.Set("a").MustRunner(noopRunner, &struct{}{})
		cs.Set("b", "b1").MustRunner(noopRunner, &struct{}{})
		cs.Set("b", "b2").MustRunner(noopRunner, &struct{}{})
		return cs
	}

	t.Run("appends to existing hooks", func(t *testing.T) {
		cs := setup(t)
		order := []string{}
		hookFirst := subcmd.PreHook(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "first")
			return ctx, nil, nil
		})
		hookSecond := subcmd.PreHook(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "second")
			return ctx, nil, nil
		})
		cs.Set("a").MustSetPreHooks(hookFirst)
		cs.Set("a").MustAppendPreHooks(hookSecond)
		if err := cs.DispatchWithArgs(ctx, "root", "a"); err != nil {
			t.Fatal(err)
		}
		if len(order) != 2 || order[0] != "first" || order[1] != "second" {
			t.Errorf("got %v, want [first second]", order)
		}
	})

	t.Run("appends to all descendants", func(t *testing.T) {
		cs := setup(t)
		callCount := 0
		hook := subcmd.PreHook(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			callCount++
			return ctx, nil, nil
		})
		cs.Set("b").MustAppendPreHooks(hook)
		if err := cs.DispatchWithArgs(ctx, "root", "b", "b1"); err != nil {
			t.Fatal(err)
		}
		if err := cs.DispatchWithArgs(ctx, "root", "b", "b2"); err != nil {
			t.Fatal(err)
		}
		if callCount != 2 {
			t.Errorf("hook called %d times, want 2", callCount)
		}
	})

	t.Run("does not affect siblings", func(t *testing.T) {
		cs := setup(t)
		callCount := 0
		hook := subcmd.PreHook(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			callCount++
			return ctx, nil, nil
		})
		cs.Set("b").MustAppendPreHooks(hook)
		if err := cs.DispatchWithArgs(ctx, "root", "a"); err != nil {
			t.Fatal(err)
		}
		if callCount != 0 {
			t.Errorf("hook called %d times for sibling 'a', want 0", callCount)
		}
	})

	t.Run("error on unknown command", func(t *testing.T) {
		cs := setup(t)
		err := cs.Set("nonexistent").AppendPreHooks()
		if err == nil || !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("expected error mentioning nonexistent, got %v", err)
		}
	})
}

func TestPreHooksOnToplevelCommand(t *testing.T) {
	// A YAML spec with no sub-commands produces a single top-level command
	// (cmdSet.cmd rather than cmdSet.cmds). Verify that Set().AppendPreHooks
	// and Set().SetPreHooks work correctly in that case.
	ctx := context.Background()

	const spec = `name: toplevel
summary: a single top-level command
`
	var order []string
	out := &strings.Builder{}

	makeHook := func(name string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "pre-"+name)
			return ctx, nil, nil
		}
	}

	setup := func(t *testing.T) *subcmd.CommandSetYAML {
		t.Helper()
		order = nil
		out.Reset()
		cs := subcmd.MustFromYAML(spec)
		cs.Set("toplevel").MustRunner((&runner{name: "toplevel", out: out}).cmd, &exampleFlags{})
		return cs
	}

	t.Run("AppendPreHooks runs hook", func(t *testing.T) {
		cs := setup(t)
		cs.Set("toplevel").MustAppendPreHooks(makeHook("a"))
		if err := cs.DispatchWithArgs(ctx, "toplevel"); err != nil {
			t.Fatal(err)
		}
		if len(order) != 1 || order[0] != "pre-a" {
			t.Errorf("got %v, want [pre-a]", order)
		}
	})

	t.Run("AppendPreHooks accumulates", func(t *testing.T) {
		cs := setup(t)
		cs.Set("toplevel").MustAppendPreHooks(makeHook("a"))
		cs.Set("toplevel").MustAppendPreHooks(makeHook("b"))
		if err := cs.DispatchWithArgs(ctx, "toplevel"); err != nil {
			t.Fatal(err)
		}
		want := []string{"pre-a", "pre-b"}
		if len(order) != len(want) {
			t.Fatalf("got %v, want %v", order, want)
		}
		for i, v := range want {
			if order[i] != v {
				t.Errorf("step %d: got %q, want %q", i, order[i], v)
			}
		}
	})

	t.Run("SetPreHooks replaces", func(t *testing.T) {
		cs := setup(t)
		cs.Set("toplevel").MustAppendPreHooks(makeHook("a"))
		cs.Set("toplevel").MustSetPreHooks(makeHook("b"))
		if err := cs.DispatchWithArgs(ctx, "toplevel"); err != nil {
			t.Fatal(err)
		}
		if len(order) != 1 || order[0] != "pre-b" {
			t.Errorf("got %v, want [pre-b]", order)
		}
	})
}

func TestErrors(t *testing.T) {
	r := &runner{name: "a", out: nil}
	cmdSet := subcmd.MustFromYAMLTemplate(oneLevel)
	var notastruct int
	err := cmdSet.Set("l0.1").Runner(r.cmd, &notastruct)
	if err == nil || err.Error() != "*int is not a pointer to a struct" {
		t.Errorf("missing or wrong error: %v", err)
	}
}

func TestArgDetail(t *testing.T) {
	cmdSet := subcmd.MustFromYAML(`name: l0
summary: documentation for l0
commands:
  - name: l0.1
    summary: summary of l0.1
    arguments:
      - <arg1> - arg 1
      - <arg2> - arg 2
  - name: l1
    summary: summary of l1
    commands:
      - name: l1.1
        summary: summary of l1.1
        arguments:
          - <arg1> - arg 1
          - ...
  - name: l2
    summary: summary of l2
    commands:
      - name: l2.1
        commands:
          - name: l2.1.1
            arguments:
              - <arg1> - arg 1
`)

	if got, want := cmdSet.Usage("l0.1"), `Usage of command "l0.1": summary of l0.1
l0.1 <arg1> <arg2>
  <arg1> - arg 1
  <arg2> - arg 2
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := cmdSet.Usage("l1/l1.1"), `Usage of command "l1.1": summary of l1.1
l1.1 <arg1> ...
  <arg1> - arg 1
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := cmdSet.Usage("l2/l2.1/l2.1.1"), `Usage of command "l2.1.1"
l2.1.1 <arg1>
  <arg1> - arg 1
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
