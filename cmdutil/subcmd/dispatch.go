// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"context"
	"errors"
	"os"

	"cloudeng.io/cmdutil"
)

var errInterrupt = errors.New("interrupt")

// Dispatch runs the supplied CommandSetYAML with support for signal handling.
// It will exit with an error if the context is cancelled with an interrupt
// signal or if the CommandSetYAML returns an error.
func Dispatch(ctx context.Context, cli *CommandSetYAML) {
	ctx, cancel := context.WithCancelCause(ctx)
	cmdutil.HandleSignals(func() { cancel(errInterrupt) }, os.Interrupt)
	err := cli.Dispatch(ctx)
	if context.Cause(ctx) == errInterrupt {
		cmdutil.Exit("%v", errInterrupt)
	}
	if err != nil {
		cmdutil.Exit("%v", err)
	}
}
