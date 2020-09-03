// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/errors"
)

type flagsA struct {
	A int `fl:"flag-a,,a: an int flag"`
	B int `fl:"flag-b,,b: an int flag"`
}

type flagsB struct {
	X string `fl:"flag-x,,x: a string flag"`
	Y string `fl:"flag-y,,y: a string flag"`
}

func TestCommandSet(t *testing.T) {
	ctx := context.Background()
	var err error
	assertNoError := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	out := &strings.Builder{}
	runnerA := func(ctx context.Context, values interface{}, args []string) error {
		fl, ok := values.(*flagsA)
		if !ok {
			t.Fatalf("wrong type: %T", values)
		}
		fmt.Fprintf(out, "%v .. %v\n", fl.A, fl.B)
		return nil
	}
	runnerB := func(ctx context.Context, values interface{}, args []string) error {
		fl, ok := values.(*flagsB)
		if !ok {
			t.Fatalf("wrong type: %T", values)
		}
		fmt.Fprintf(out, "%v .. %v\n", fl.X, fl.Y)
		return nil
	}

	cmdAFlags := subcmd.NewFlags("cmd-a", "subcmd a", "<args>...")
	err = cmdAFlags.RegisterFlagStruct("fl", &flagsA{}, nil, nil)
	assertNoError()
	cmdBFlags := subcmd.NewFlags("cmd-b", "subcmd b")
	err = cmdBFlags.RegisterFlagStruct("fl", &flagsB{}, nil, nil)
	assertNoError()
	commands := subcmd.First(cmdAFlags, runnerA).
		Append(cmdBFlags, runnerB)

	err = commands.DispatchWithArgs(ctx, "cmd-a", "--flag-a=1", "--flag-b=3")
	assertNoError()
	if got, want := out.String(), "1 .. 3\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := commands.Defaults(), `cmd-a: subcmd a
cmd-a [--flag-a=0 --flag-b=0] <args>...
  -flag-a int
    	a: an int flag
  -flag-b int
    	b: an int flag

cmd-b: subcmd b
cmd-b [--flag-x= --flag-y=]
  -flag-x string
    	x: a string flag
  -flag-y string
    	y: a string flag

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out.Reset()
	err = commands.DispatchWithArgs(ctx, "cmd-b", "--flag-x=s1", "--flag-y=s3")
	assertNoError()
	if got, want := out.String(), "s1 .. s3\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestCommandOptions(t *testing.T) {
	ctx := context.Background()

	numArgs := -1
	runnerA := func(ctx context.Context, values interface{}, args []string) error {
		if _, ok := values.(*flagsA); !ok {
			t.Fatalf("wrong type: %T", values)
		}
		numArgs = len(args)
		return nil
	}
	cmdAFlags := subcmd.NewFlags("cmd", "subcmd", "<args>...")
	err := cmdAFlags.RegisterFlagStruct("fl", &flagsA{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	cmdExactlyN := subcmd.First(cmdAFlags, runnerA, subcmd.ExactlyNumArguments(2))
	cmdNoArgs := subcmd.First(cmdAFlags, runnerA, subcmd.WithoutArguments())
	cmdOptional := subcmd.First(cmdAFlags, runnerA, subcmd.OptionalSingleArgument())

	expectedError := func(errmsg string) {
		if err == nil || !strings.Contains(err.Error(), errmsg) {
			t.Errorf("missing or incorrect error: %v does not contain %v", err, errmsg)
		}
	}
	expectedNArgs := func(n int) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := numArgs, n; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		numArgs = -1
	}
	err = cmdExactlyN.DispatchWithArgs(ctx, "cmd")
	expectedError("cmd: accepts exactly 2 arguments")
	err = cmdExactlyN.DispatchWithArgs(ctx, "cmd", "a", "b")
	expectedNArgs(2)
	err = cmdNoArgs.DispatchWithArgs(ctx, "cmd", "aaa")
	expectedError("cmd: does not accept any arguments")
	err = cmdNoArgs.DispatchWithArgs(ctx, "cmd")
	expectedNArgs(0)
	err = cmdOptional.DispatchWithArgs(ctx, "cmd", "a", "b")
	expectedError("cmd: accepts at most one argument")
	err = cmdOptional.DispatchWithArgs(ctx, "cmd")
	expectedNArgs(0)
	err = cmdOptional.DispatchWithArgs(ctx, "cmd", "a")
	expectedNArgs(1)
}

func TestMultiLevel(t *testing.T) {
	ctx := context.Background()

	cmd1, cmd2, cmd12, cmd22 := false, false, false, false

	c1 := func(ctx context.Context, values interface{}, args []string) error {
		cmd1 = true
		return nil
	}

	c2 := func(ctx context.Context, values interface{}, args []string) error {
		cmd2 = true
		return nil
	}

	c12 := func(ctx context.Context, values interface{}, args []string) error {
		cmd12 = true
		return nil
	}

	c22 := func(ctx context.Context, values interface{}, args []string) error {
		cmd22 = true
		return nil
	}

	c1Flags := subcmd.NewFlags("c1", "c1")
	c2Flags := subcmd.NewFlags("c2", "c2")
	c12Flags := subcmd.NewFlags("c12", "c12")
	c22Flags := subcmd.NewFlags("c22", "c22")
	errs := errors.M{}
	errs.Append(c1Flags.RegisterFlagStruct("fl", &flagsA{}, nil, nil))
	errs.Append(c2Flags.RegisterFlagStruct("fl", &flagsA{}, nil, nil))
	errs.Append(c12Flags.RegisterFlagStruct("fl", &flagsA{}, nil, nil))
	if err := errs.Err(); err != nil {
		t.Fatal(err)
	}

	l2 := subcmd.First(c12Flags, c12).Append(c22Flags, c22)
	l1 := subcmd.First(c1Flags, c1, subcmd.SubCommands(l2)).
		Append(c2Flags, c2)

	if got, want := l1.Defaults(), `c1: c1
c1 [--flag-a=0 --flag-b=0] c12|c22 ...
  -flag-a int
    	a: an int flag
  -flag-b int
    	b: an int flag

c2: c2
c2 [--flag-a=0 --flag-b=0]
  -flag-a int
    	a: an int flag
  -flag-b int
    	b: an int flag

`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var err error

	assert := func(b bool) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b {
			t.Errorf("expected value to be true")
		}
	}

	err = l1.DispatchWithArgs(ctx, "c1", "c12")
	assert(cmd12)
	assert(!cmd22)
	assert(!cmd1)
	assert(!cmd2)

	err = l1.DispatchWithArgs(ctx, "c1", "c22")
	assert(cmd12)
	assert(cmd22)
	assert(!cmd1)
	assert(!cmd2)

	err = l1.DispatchWithArgs(ctx, "c2")
	assert(cmd12)
	assert(cmd22)
	assert(!cmd1)
	assert(cmd2)

	err = l1.DispatchWithArgs(ctx, "c1")
	if err == nil || !strings.Contains(err.Error(), "missing top level command: available commands are: c12, c22") {
		t.Errorf("expected an error: %v", err)
	}

	err = l1.DispatchWithArgs(ctx, "c1", "cx")
	if err == nil || !strings.Contains(err.Error(), "cx is not one of the supported commands: c12, c22") {
		t.Errorf("expected an error: %v", err)
	}
}
