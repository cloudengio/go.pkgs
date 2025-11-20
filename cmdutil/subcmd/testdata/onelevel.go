// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
)

var cmdSet *subcmd.CommandSet

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
	cmdSet = subcmd.NewCommandSet(l0_1, l0_2)
	cmdSet.Document("describe l0")
}

func l0_1(ctx context.Context, values any, args []string) error {
	fv := values.(*exampleFlags)
	fmt.Printf("l0_1: flag value: %v\n", fv.Flag1)
	return nil
}

func l0_2(ctx context.Context, values any, args []string) error {
	fv := values.(*exampleFlags)
	fmt.Printf("l0_2: flag value: %v\n", fv.Flag1)
	return nil
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
