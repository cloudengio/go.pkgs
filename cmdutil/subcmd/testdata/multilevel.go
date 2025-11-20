// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
)

var (
	cmdSet      *subcmd.CommandSet
	globalFlags GlobalFlags
)

type GlobalFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
}

type exampleFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
}

func init() {
	l0_1 := subcmd.NewCommand(
		"l0.1",
		subcmd.MustRegisteredFlagSet(&exampleFlags{}),
		l0_1,
		subcmd.ExactlyNumArguments(2))
	l0_1.Document("summary of l0.1")
	l0_2 := subcmd.NewCommand(
		"l0.2",
		subcmd.MustRegisteredFlagSet(&exampleFlags{}),
		l0_2,
		subcmd.AtLeastNArguments(1))
	l0_2.Document("summary of l0.2")

	l1_1 := subcmd.NewCommand(
		"l1.1",
		subcmd.MustRegisteredFlagSet(&exampleFlags{}),
		l1_1,
		subcmd.WithoutArguments())
	l1_1.Document("describe l1.1")

	l1_2 := subcmd.NewCommand(
		"l1.2",
		subcmd.MustRegisteredFlagSet(&exampleFlags{}),
		l1_2,
		subcmd.WithoutArguments())
	l1_2.Document("describe l1.2")

	l1 := subcmd.NewCommandLevel("l1", subcmd.NewCommandSet(l1_1, l1_2))
	l1.Document("summary of l1")

	cmdSet = subcmd.NewCommandSet(l0_1, l0_2, l1)
	cmdSet.Document("summary of l0")

	gfs := subcmd.GlobalFlagSet().MustRegisterFlagStruct(&globalFlags, nil, nil)
	cmdSet.WithGlobalFlags(gfs)
	cmdSet.WithMain(mainWrapper)
}

func runner(ctx context.Context, name string, values any, args []string) error {
	fmt.Printf("%s: flag value: %v\n", name, values.(*exampleFlags).Flag1)
	return nil
}

func l0_1(ctx context.Context, values any, args []string) error {
	return runner(ctx, "l0_1", values, args)
}

func l0_2(ctx context.Context, values any, args []string) error {
	return runner(ctx, "l0_2", values, args)
}

func l1_1(ctx context.Context, values any, args []string) error {
	return runner(ctx, "l1_1", values, args)
}

func l1_2(ctx context.Context, values any, args []string) error {
	return runner(ctx, "l1_1", values, args)
}

func mainWrapper(ctx context.Context, cmdRunner func(ctx context.Context) error) error {
	fmt.Printf("main wrapper: ")
	return cmdRunner(ctx)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
