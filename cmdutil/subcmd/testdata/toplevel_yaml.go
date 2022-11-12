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
summary: overall documentation
  for the simple command
`

var cmdSet *subcmd.CommandSetYAML = subcmd.MustFromYAML(simple)

type simpleFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
	Flag2 int `subcmd:"flag2,2,flag2"`
}

func init() {
	cmdSet.Set("simple").RunnerAndFlags(runner, subcmd.MustRegisteredFlagSet(&simpleFlags{}))
}

func runner(ctx context.Context, values interface{}, args []string) error {
	fv := values.(*simpleFlags)
	fmt.Printf("runner: flag values: %v %v\n", fv.Flag1, fv.Flag2)
	return nil
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
