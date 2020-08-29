// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package subcmd provides a simple, single-level, sub-command facility.
// It allows for creating single-level command trees of the form:
//
// Usage of <tool>
//   <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
//   ...
//   <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>
package subcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"cloudeng.io/cmdutil/flags"
)

// Flags represents the name, description and flags for a single
// top-level command.
type Flags struct {
	flagSet     *flag.FlagSet
	description string
	sm          *flags.SetMap
}

// NewFlags returns a new instance of Flags.
func NewFlags(name, description string) *Flags {
	return &Flags{
		flagSet:     flag.NewFlagSet(name, flag.ContinueOnError),
		description: description,
	}
}

// RegisterFlagStruct registers a struct, using RegisterFlagsInStructWithSetMap.
// The returned SetMap can be queried by the IsSet method.
func (cf *Flags) RegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error {
	sm, err := flags.RegisterFlagsInStructWithSetMap(cf.flagSet, tag, structWithFlags, valueDefaults, usageDefaults)
	cf.sm = sm
	return err
}

// IsSet returns true if the supplied flag variable's value has been
// set, either via a string literal in the struct or via the valueDefaults
// argument to RegisterFlagStruct.
func (cf *Flags) IsSet(field interface{}) (string, bool) {
	return cf.IsSet(field)
}

// Runner is the type of the function to be called to run a single
// top-level command.
type Runner func(ctx context.Context, args []string) error

// Command represents a single top level command.
type Command struct {
	flags  *Flags
	opts   options
	runner Runner
}

// Commands provides a simple implementation of structured command line
// processing that supports a single level of sub-commands each with their
// own set of flags.
type Commands []*Command

// CommandOption represents an option controlling the handling of a given
// command.
type CommandOption func(*options)

type options struct {
	withoutArgs bool
}

// WithoutArguments specifies that the command takes no arguments.
func WithoutArguments() CommandOption {
	return func(o *options) {
		o.withoutArgs = true
	}
}

// Append adds a new top level command.
func (c Commands) Append(flags *Flags, runner Runner, options ...CommandOption) Commands {
	cmd := &Command{flags: flags, runner: runner}
	for _, fn := range options {
		fn(&cmd.opts)
	}
	return append(c, cmd)
}

// Defaults returns the value of Defaults for each command in commands.
func (c Commands) Defaults() string {
	out := &strings.Builder{}
	for _, cmd := range c {
		fs := cmd.flags.flagSet
		fmt.Fprintf(out, "%v: %v\n", fs.Name(), cmd.flags.description)
		out.WriteString(flags.Defaults(fs))
		out.WriteString("\n")
	}
	return out.String()
}

// Usage returns a function that can be assigned to flag.Usage.
func (c Commands) Usage(name string) func() {
	return func() {
		fmt.Printf("Usage of %v\n%s\n", name, c.Defaults())
	}
}

// Commands returns the list of available commands.
func (c Commands) Commands() []string {
	out := make([]string, len(c))
	for i, cmd := range c {
		out[i] = cmd.flags.flagSet.Name()
	}
	return out
}

// Dispatch determines which top level command has been requested, if any,
// parses the command line appropriately and then runs its associated function.
func (c Commands) Dispatch(ctx context.Context) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("missing top level command: available commands are: %v", strings.Join(c.Commands(), ", "))
	}
	requested := os.Args[1]
	for _, cmd := range c {
		fs := cmd.flags.flagSet
		if requested == fs.Name() {
			if cmd.runner == nil {
				return fmt.Errorf("no runner registerd with %v", requested)
			}
			args := os.Args[2:]
			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("%v: failed to parse flags: %v", fs.Name(), err)
			}
			args = fs.Args()
			if cmd.opts.withoutArgs && len(args) > 0 {
				return fmt.Errorf("%v: does not accept any arguments", fs.Name())
			}
			return cmd.runner(ctx, args)
		}
	}
	return fmt.Errorf("%v is not one of the supported commands: %v", requested, strings.Join(c.Commands(), ", "))
}
