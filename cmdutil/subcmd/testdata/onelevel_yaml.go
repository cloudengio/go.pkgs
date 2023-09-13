// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
)

const onelevel = `name: l0
summary: describe l0
commands:
  - name: l0.1
    summary: summary of l0.1
    arguments:
        - <arg1>
        - <arg2>
  - name: l0.2
    summary: summary of l0.2
    arguments:
      - <arg1>
      - ...
`

var cmdSet *subcmd.CommandSetYAML = subcmd.MustFromYAML(onelevel)

type exampleFlags struct {
	Flag1 int `subcmd:"flag1,1,flag1"`
}

func init() {
	cmdSet.Set("l0.1").Runner(l0_1, &exampleFlags{})
	cmdSet.Set("l0.2").Runner(l0_2, &exampleFlags{})
}

func l0_1(ctx context.Context, values interface{}, args []string) error {
	fv := values.(*exampleFlags)
	fmt.Printf("l0_1: flag value: %v\n", fv.Flag1)
	return nil
}

func l0_2(ctx context.Context, values interface{}, args []string) error {
	fv := values.(*exampleFlags)
	fmt.Printf("l0_2: flag value: %v\n", fv.Flag1)
	return nil
}

func main() {
	cmdSet.MustDispatch(context.Background())
}
