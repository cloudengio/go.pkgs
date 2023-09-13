// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
)

var globalFlags GlobalFlags

type GlobalFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
}

type exampleFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
}

const multilevel = `name: l0
summary: summary of l0
commands:
  - name: l0.1
    summary: summary of l0.1
    arguments:
        - <arg1>
        - <arg2>
  - name: l0.2
    summary: summary of l0.2
  - name: l1
    summary: summary of l1
    commands:
      - name: l1.1
        summary: describe l1.1
      - name: l1.2
        summary: describe l1.2
`

var cmdSet *subcmd.CommandSetYAML = subcmd.MustFromYAML(multilevel)

func init() {
	cmdSet.Set("l0.1").Runner(l0_1, &exampleFlags{})
	cmdSet.Set("l0.2").Runner(l0_2, &exampleFlags{})

	cmdSet.Set("l1", "l1.1").Runner(l1_1, &exampleFlags{})
	cmdSet.Set("l1", "l1.2").Runner(l1_2, &exampleFlags{})

	gfs := subcmd.GlobalFlagSet().MustRegisterFlagStruct(&globalFlags, nil, nil)
	cmdSet.WithGlobalFlags(gfs)
	cmdSet.WithMain(mainWrapper)
}

func runner(ctx context.Context, name string, values interface{}, args []string) error {
	fmt.Printf("%s: flag value: %v\n", name, values.(*exampleFlags).Flag1)
	return nil
}

func l0_1(ctx context.Context, values interface{}, args []string) error {
	return runner(ctx, "l0_1", values, args)
}

func l0_2(ctx context.Context, values interface{}, args []string) error {
	return runner(ctx, "l0_2", values, args)
}

func l1_1(ctx context.Context, values interface{}, args []string) error {
	return runner(ctx, "l1_1", values, args)
}

func l1_2(ctx context.Context, values interface{}, args []string) error {
	return runner(ctx, "l1_1", values, args)
}

func mainWrapper(ctx context.Context, cmdRunner func(ctx context.Context) error) error {
	fmt.Printf("main wrapper: ")
	return cmdRunner(ctx)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
