// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package subcmd provides a simple sub-command facility.
// It allows for creating command trees of the form:
//
// Usage of <tool>
//   <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
//      <sub-command-2-1> <flags for sub-command-2-1> <args for sub-comand-2-1>
//      ...
//      <sub-command-2-2> <flags for sub-command-2-2> <args for sub-comand-2-2>
//   ...
//   <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>
//
// Creating a command consists of defining the flags with associated
// descriptions and then associating the newly create Flags struct with
// the function to be run to implement that command.
// The flags.RegisterFlagsInStruct paradigm is used for defining flags.
// A CommandSet is then created as follows:
//
//    cmds = subcmd.First(...).Append(...).Append(...)
//
// Once created, typically in an init function, the CommandSet may be used
// by calling its Dispatch or DispatchWithArgs methods typically from
// the main function.
//
// The encapsulation of all the flag definitions within a struct and then
// making that struct available to the runner function as a parameter
// conveniently avoids having to define flag values at a global level.
//
// Note that this package will never call flag.Parse and will not associate
// any flags with flag.CommandLine. Commands.Usage() can be used to set
// flag.Usage.
package subcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloudeng.io/cmdutil/flags"
)

// Flags represents the name, description and flags for a command.
type Flags struct {
	flagSet     *flag.FlagSet
	flagValues  interface{}
	description string
	arguments   string
	sm          *flags.SetMap
}

// NewFlags returns a new instance of Flags. The flags are used to name
// and describe the command they are associated with and optionally,
// the arguments may also be given a desription.
func NewFlags(name, description string, arguments ...string) *Flags {
	return &Flags{
		flagSet:     flag.NewFlagSet(name, flag.ContinueOnError),
		description: description,
		arguments:   strings.Join(arguments, " "),
	}
}

// RegisterFlagStruct registers a struct, using RegisterFlagsInStructWithSetMap.
// The returned SetMap can be queried by the IsSet method.
func (cf *Flags) RegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error {
	sm, err := flags.RegisterFlagsInStructWithSetMap(cf.flagSet, tag, structWithFlags, valueDefaults, usageDefaults)
	cf.sm = sm
	cf.flagValues = structWithFlags
	return err
}

// MustRegisterFlagStruct is like RegisterFlagStruct except that it panics
// on encountering an error. Its use is encouraged over RegisterFlagStruct from
// within init functions.
func (cf *Flags) MustRegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) {
	err := cf.RegisterFlagStruct(tag, structWithFlags, valueDefaults, usageDefaults)
	if err != nil {
		panic(err)
	}
}

// IsSet returns true if the supplied flag variable's value has been
// set, either via a string literal in the struct or via the valueDefaults
// argument to RegisterFlagStruct.
func (cf *Flags) IsSet(field interface{}) (string, bool) {
	return cf.IsSet(field)
}

// Runner is the type of the function to be called to run a particular command.
type Runner func(ctx context.Context, flagValues interface{}, args []string) error

// Command represents a single command.
type Command struct {
	flags  *Flags
	opts   options
	runner Runner
}

// Usage returns a string containing a 'usage' message for the command. It
// includes a summary of the command, its flags and arguments and the flag defaults.
func (cmd Command) Usage() string {
	out := &strings.Builder{}
	fs := cmd.flags.flagSet
	fmt.Fprintf(out, "%v: %v\n", fs.Name(), cmd.flags.description)
	cl := flags.NamesAndDefault(fs)
	out.WriteString(cl)
	if sc := cmd.opts.subcmds; sc != nil {
		fmt.Fprintf(out, " %v ...", strings.Join(sc.Commands(), "|"))
	} else {
		if args := cmd.flags.arguments; len(args) > 0 {
			if len(cl) > 0 {
				out.WriteString(" ")
			}
			out.WriteString(args)
		}
	}
	out.WriteString("\n")
	out.WriteString(flags.Defaults(fs))
	out.WriteString("\n")
	return out.String()
}

// CommandSet represents a set of commands that are peers to each other,
// that is, the command line must specificy one of them.
type CommandSet []*Command

// CommandOption represents an option controlling the handling of a given
// command.
type CommandOption func(*options)

type options struct {
	withoutArgs       bool
	optionalSingleArg bool
	exactArgs         bool
	numArgs           int
	subcmds           CommandSet
}

// WithoutArguments specifies that the command takes no arguments.
func WithoutArguments() CommandOption {
	return func(o *options) {
		o.withoutArgs = true
	}
}

// OptionalSingleArg specifies that the command takes an optional single argument.
func OptionalSingleArgument() CommandOption {
	return func(o *options) {
		o.optionalSingleArg = true
	}
}

// ExactlyNumArguments specifies that the command takes exactly the specified
// number of arguments.
func ExactlyNumArguments(n int) CommandOption {
	return func(o *options) {
		o.exactArgs = true
		o.numArgs = n
	}
}

// SubCommands associates a set of commands that are subordinate to the
// current one. Once all flags for the current command processed the
// first argument must be one of these commands.
func SubCommands(cmds CommandSet) CommandOption {
	return func(o *options) {
		*o = options{}
		o.subcmds = cmds
	}
}

// First creates the first command.
func First(flags *Flags, runner Runner, options ...CommandOption) CommandSet {
	var cmds CommandSet
	return cmds.Append(flags, runner, options...)
}

// Append adds a new command.
func (cmds CommandSet) Append(flags *Flags, runner Runner, options ...CommandOption) CommandSet {
	cmd := &Command{flags: flags, runner: runner}
	for _, fn := range options {
		fn(&cmd.opts)
	}
	cmd.flags.flagSet.Usage = func() {
		fmt.Fprint(cmd.flags.flagSet.Output(), cmd.Usage())
	}
	return append(cmds, cmd)
}

// Defaults returns the value of Defaults for each command in commands.
func (cmds CommandSet) Defaults() string {
	out := &strings.Builder{}
	for _, cmd := range cmds {
		out.WriteString(cmd.Usage())
	}
	return out.String()
}

// Usage returns a function that can be assigned to flag.Usage.
func (cmds CommandSet) Usage(name string) func() {
	return func() {
		fmt.Printf("Usage of %v\n\n%s\n", name, cmds.Defaults())
	}
}

// Commands returns the list of available commands.
func (cmds CommandSet) Commands() []string {
	out := make([]string, len(cmds))
	for i, cmd := range cmds {
		out[i] = cmd.flags.flagSet.Name()
	}
	return out
}

func (cmds CommandSet) Dispatch(ctx context.Context) error {
	return cmds.DispatchWithArgs(ctx, os.Args[1:]...)
}

// Dispatch determines which top level command has been requested, if any,
// parses the command line appropriately and then runs its associated function.
func (cmds CommandSet) DispatchWithArgs(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		cmds.Usage(filepath.Base(os.Args[0]))()
		return fmt.Errorf("missing top level command: available commands are: %v", strings.Join(cmds.Commands(), ", "))
	}
	requested := args[0]
	switch requested {
	case "help", "-help", "--help", "-h", "--h":
		cmds.Usage(filepath.Base(os.Args[0]))()
		return flag.ErrHelp
	}
	for _, cmd := range cmds {
		fs := cmd.flags.flagSet
		if requested == fs.Name() {
			if cmd.runner == nil {
				return fmt.Errorf("no runner registered for %v", requested)
			}
			args := args[1:]
			if err := fs.Parse(args); err != nil {
				if err == flag.ErrHelp {
					return err
				}
				return fmt.Errorf("%v: failed to parse flags: %v", fs.Name(), err)
			}
			args = fs.Args()
			switch {
			case cmd.opts.withoutArgs:
				if len(args) > 0 {
					return fmt.Errorf("%v: does not accept any arguments", fs.Name())
				}
			case cmd.opts.optionalSingleArg:
				if len(args) > 1 {
					return fmt.Errorf("%v: accepts at most one argument", fs.Name())
				}
			case cmd.opts.exactArgs:
				if len(args) != cmd.opts.numArgs {
					return fmt.Errorf("%v: accepts exactly %v arguments", fs.Name(), cmd.opts.numArgs)
				}
			case cmd.opts.subcmds != nil:
				return cmd.opts.subcmds.DispatchWithArgs(ctx, args...)
			}
			return cmd.runner(ctx, cmd.flags.flagValues, args)
		}
	}
	return fmt.Errorf("%v is not one of the supported commands: %v", requested, strings.Join(cmds.Commands(), ", "))
}
