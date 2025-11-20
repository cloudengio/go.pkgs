// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
)

const simple = `name: simple
usage: 
`

var cmdSet *subcmd.CommandSet

type simpleFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
	Flag2 int `subcmd:"flag2,2,flag2"`
}

func init() {
	toplevel := subcmd.NewCommand(
		"simple",
		subcmd.MustRegisteredFlagSet(&simpleFlags{}),
		runner,
		subcmd.WithoutArguments())
	toplevel.Document("overall documentation for the simple command")
	cmdSet = subcmd.NewCommandSet()
	cmdSet.TopLevel(toplevel)
}

func runner(ctx context.Context, values any, args []string) error {
	fv := values.(*simpleFlags)
	fmt.Printf("runner: flag values: %v %v\n", fv.Flag1, fv.Flag2)
	return nil
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
