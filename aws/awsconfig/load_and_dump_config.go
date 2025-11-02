// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

// This command is provided primarily for debugging configuration issues.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/cmdutil/flags"
)

var cl awsconfig.AWSFlags

func main() {
	err := flags.RegisterFlagsInStruct(flag.CommandLine, "subcmd", &cl, nil, nil)
	if err != nil {
		panic(err)
	}
	flag.Parse()
	ctx := context.Background()
	cfg, err := awsconfig.LoadUsingFlags(ctx, cl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	if err := awsconfig.DebugPrintConfig(ctx, os.Stdout, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
