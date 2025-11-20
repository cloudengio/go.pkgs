// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"context"
	"flag"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/errors"
)

func ExampleCommandSet() {
	ctx := context.Background()
	type globalFlags struct {
		Verbosity int `subcmd:"v,0,debugging verbosity"`
	}
	var globalValues globalFlags
	type rangeFlags struct {
		From int `subcmd:"from,1,start value for a range"`
		To   int `subcmd:"to,2,end value for a range "`
	}
	printRange := func(_ context.Context, values any, _ []string) error {
		r := values.(*rangeFlags)
		fmt.Printf("%v: %v..%v\n", globalValues.Verbosity, r.From, r.To)
		return nil
	}

	fs := subcmd.MustRegisterFlagStruct(&rangeFlags{}, nil, nil)
	// Subcommands are added using subcmd.NewCommandLevel.
	cmd := subcmd.NewCommand("ranger", fs, printRange, subcmd.WithoutArguments())
	cmd.Document("print an integer range")
	cmdSet := subcmd.NewCommandSet(cmd)
	globals := subcmd.NewFlagSet()
	globals.MustRegisterFlagStruct(&globalValues, nil, nil)
	cmdSet.WithGlobalFlags(globals)

	// Use cmdSet.Dispatch to access os.Args.
	fmt.Printf("%s", cmdSet.Usage("example-command"))
	fmt.Printf("%s", cmdSet.Defaults("example-command"))
	if err := cmdSet.DispatchWithArgs(ctx, "example-command", "ranger"); err != nil {
		panic(err)
	}
	if err := cmdSet.DispatchWithArgs(ctx, "example-command", "-v=3", "ranger", "--from=10", "--to=100"); err != nil {
		panic(err)
	}

	// Output:
	// Usage of example-command
	//
	//  ranger - print an integer range
	// Usage of example-command
	//
	//  ranger - print an integer range
	//
	// global flags: [--v=0]
	//   -v int
	//     debugging verbosity
	//
	// 0: 1..2
	// 3: 10..100
}

type flagsA struct {
	A int `subcmd:"flag-a,,a: an int flag"`
	B int `subcmd:"flag-b,,b: an int flag"`
}

type flagsB struct {
	X string `subcmd:"flag-x,,x: a string flag"`
	Y string `subcmd:"flag-y,,y: a string flag"`
}

func TestCommandSet(t *testing.T) {
	ctx := context.Background()
	var err error
	assertNoError := func() {
		_, _, line, _ := runtime.Caller(1)
		if err != nil {
			t.Fatalf("line %v: %v", line, err)
		}
	}

	out := &strings.Builder{}
	runnerA := func(_ context.Context, values any, _ []string) error {
		fl, ok := values.(*flagsA)
		if !ok {
			t.Fatalf("wrong type: %T", values)
		}
		fmt.Fprintf(out, "%v .. %v\n", fl.A, fl.B)
		return nil
	}
	runnerB := func(_ context.Context, values any, _ []string) error {
		fl, ok := values.(*flagsB)
		if !ok {
			t.Fatalf("wrong type: %T", values)
		}
		fmt.Fprintf(out, "%v .. %v\n", fl.X, fl.Y)
		return nil
	}

	cmdAFlags, err := subcmd.RegisterFlagStruct(&flagsA{}, nil, nil)
	assertNoError()
	cmdA := subcmd.NewCommand("cmd-a", cmdAFlags, runnerA)
	cmdA.Document("subcmd a", "<args>...")

	cmdBFlags, err := subcmd.RegisterFlagStruct(&flagsB{}, nil, nil)
	assertNoError()
	cmdB := subcmd.NewCommand("cmd-b", cmdBFlags, runnerB)
	cmdB.Document("subcmd b", "")
	commands := subcmd.NewCommandSet(cmdA, cmdB)
	commands.Document("an explanation which will be line wrapped to something sensible and approaching 80 chars")

	err = commands.DispatchWithArgs(ctx, "test", "cmd-a", "--flag-a=1", "--flag-b=3")
	assertNoError()
	if got, want := out.String(), "1 .. 3\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := commands.Usage("binary"), `Usage of binary

 an explanation which will be line wrapped to something sensible and approaching
 80 chars

 cmd-a - subcmd a
 cmd-b - subcmd b
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out.Reset()
	err = commands.DispatchWithArgs(ctx, "test", "cmd-b", "--flag-x=s1", "--flag-y=s3")
	if err != nil {
		t.Fatal(err)
	}
	assertNoError()
	if got, want := out.String(), "s1 .. s3\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	errHelp := func() {
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp: got %v", err)
		}
	}
	help := &strings.Builder{}
	commands.SetOutput(help)
	err = commands.DispatchWithArgs(ctx, "test", "help", "cmd-a")
	errHelp()
	if got, want := help.String(), `Usage of command "cmd-a": subcmd a
cmd-a [--flag-a=0 --flag-b=0] <args>...

  -flag-a int
    a: an int flag
  -flag-b int
    b: an int flag


`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCommandOptions(t *testing.T) {
	ctx := context.Background()

	numArgs := -1
	runnerA := func(_ context.Context, values any, args []string) error {
		if _, ok := values.(*flagsA); !ok {
			t.Fatalf("wrong type: %T", values)
		}
		numArgs = len(args)
		return nil
	}
	cmdAFlags := subcmd.NewFlagSet()
	err := cmdAFlags.RegisterFlagStruct(&flagsA{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	cmdset := subcmd.NewCommandSet(
		subcmd.NewCommand("exactly-two", cmdAFlags, runnerA, subcmd.ExactlyNumArguments(2)),
		subcmd.NewCommand("none", cmdAFlags, runnerA, subcmd.WithoutArguments()),
		subcmd.NewCommand("at-most-one", cmdAFlags, runnerA, subcmd.OptionalSingleArgument()),
		subcmd.NewCommand("at-least-two", cmdAFlags, runnerA, subcmd.AtLeastNArguments(2)),
	)

	expectedError := func(errmsg string) {
		_, _, line, _ := runtime.Caller(1)
		if err == nil || !strings.Contains(err.Error(), errmsg) {
			t.Errorf("line %v: missing or incorrect error: %v does not contain %v", line, err, errmsg)
		}
	}
	expectedNArgs := func(n int) {
		_, _, line, _ := runtime.Caller(1)
		if err != nil {
			t.Fatalf("line %v: unexpected error: %v", line, err)
		}
		if got, want := numArgs, n; got != want {
			t.Errorf("line %v: got %v, want %v", line, got, want)
		}
		numArgs = -1
	}
	err = cmdset.DispatchWithArgs(ctx, "test", "exactly-two")
	expectedError("exactly-two: accepts exactly 2 arguments")
	err = cmdset.DispatchWithArgs(ctx, "test", "at-least-two")
	expectedError("at-least-two: accepts at least 2 arguments")
	err = cmdset.DispatchWithArgs(ctx, "test", "exactly-two", "a", "b")
	expectedNArgs(2)
	err = cmdset.DispatchWithArgs(ctx, "test", "none", "aaa")
	expectedError("none: does not accept any arguments")
	err = cmdset.DispatchWithArgs(ctx, "test", "none")
	expectedNArgs(0)
	err = cmdset.DispatchWithArgs(ctx, "test", "at-most-one", "a", "b")
	expectedError("at-most-one: accepts at most one argument")
	err = cmdset.DispatchWithArgs(ctx, "test", "at-most-one")
	expectedNArgs(0)
	err = cmdset.DispatchWithArgs(ctx, "test", "at-most-one", "a")
	expectedNArgs(1)
	err = cmdset.DispatchWithArgs(ctx, "test", "at-least-two", "a", "b", "c")
	expectedNArgs(3)
}

func TestMultiLevel(t *testing.T) {
	ctx := context.Background()

	type globalFlags struct {
		Verbosity int `subcmd:"v,0,debugging verbosity"`
	}
	var (
		globalValues              globalFlags
		cmd2, cmd11, cmd12, cmd31 bool
		l1Main, l2Main            bool
	)

	c2 := func(_ context.Context, _ any, _ []string) error {
		cmd2 = true
		return nil
	}

	c11 := func(_ context.Context, _ any, _ []string) error {
		cmd11 = true
		return nil
	}

	c12 := func(_ context.Context, _ any, _ []string) error {
		cmd12 = true
		return nil
	}

	c31 := func(_ context.Context, _ any, _ []string) error {
		cmd31 = true
		return nil
	}

	c1Flags := subcmd.NewFlagSet()
	c2Flags := subcmd.NewFlagSet()
	c3Flags := subcmd.NewFlagSet()
	c11Flags := subcmd.NewFlagSet()
	c12Flags := subcmd.NewFlagSet()
	c31Flags := subcmd.NewFlagSet()
	globals := subcmd.NewFlagSet()
	errs := errors.M{}
	errs.Append(c1Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(c2Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(c3Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(c11Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(c12Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(c31Flags.RegisterFlagStruct(&flagsA{}, nil, nil))
	errs.Append(globals.RegisterFlagStruct(&globalValues, nil, nil))

	if err := errs.Err(); err != nil {
		t.Fatal(err)
	}

	c11Cmd := subcmd.NewCommand("c11", c11Flags, c11)
	c11Cmd.Document("c11")
	c12Cmd := subcmd.NewCommand("c12", c12Flags, c12)
	c12Cmd.Document("c12")
	l21 := subcmd.NewCommandSet(c11Cmd, c12Cmd)
	l21.WithMain(func(ctx context.Context, runner func(context.Context) error) error {
		l2Main = true
		return runner(ctx)
	})
	c31Cmd := subcmd.NewCommand("c31", c31Flags, c31)
	c31Cmd.Document("c31")
	l31 := subcmd.NewCommandSet(c31Cmd)

	c1Cmd := subcmd.NewCommandLevel("c1", l21)
	c1Cmd.Document("c1")
	c2Cmd := subcmd.NewCommand("c2", c2Flags, c2)
	c2Cmd.Document("c2")
	c3Cmd := subcmd.NewCommandLevel("c3", l31)
	c3Cmd.Document("c3")
	l1 := subcmd.NewCommandSet(c1Cmd, c2Cmd, c3Cmd)
	l1.WithGlobalFlags(globals)
	l1.WithMain(func(ctx context.Context, runner func(context.Context) error) error {
		l1Main = true
		return runner(ctx)
	})

	if got, want := l1.Defaults("test"), `Usage of test

 c1 - c1
 c2 - c2
 c3 - c3

global flags: [--v=0]
  -v int
    debugging verbosity

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var err error

	assert := func(vals ...bool) {
		_, _, line, _ := runtime.Caller(1)
		if err != nil {
			t.Fatalf("line %v: unexpected error: %v", line, err)
		}
		for i, b := range vals {
			if !b {
				t.Errorf("line %v: val %v: expected value to be true", line, i)
			}
		}
	}

	err = l1.DispatchWithArgs(ctx, "test", "c1", "c11")
	assert(l2Main, cmd11)
	assert(!l1Main, !cmd12, !cmd2, !cmd31)

	l2Main, l1Main = false, false
	err = l1.DispatchWithArgs(ctx, "test", "c1", "c12")
	assert(l2Main, cmd11, cmd12)
	assert(!l1Main, !cmd2, !cmd31)

	l2Main, l1Main = false, false
	err = l1.DispatchWithArgs(ctx, "test", "c2")
	assert(l1Main, cmd11, cmd12)
	assert(!l2Main, cmd2, !cmd31)

	l2Main, l1Main = false, false
	err = l1.DispatchWithArgs(ctx, "test", "c3", "c31")
	assert(l1Main, cmd11, cmd12, cmd31)
	assert(!l2Main, cmd2)

	err = l1.DispatchWithArgs(ctx, "test", "c1")
	if err == nil || !strings.Contains(err.Error(), "no command specified") {
		t.Errorf("expected a particular error: %v", err)
	}

	err = l1.DispatchWithArgs(ctx, "test", "c1", "cx")
	if err == nil || !strings.Contains(err.Error(), "cx is not one of the supported commands: c11, c12") {
		t.Errorf("expected a particular error: %v", err)
	}
}
