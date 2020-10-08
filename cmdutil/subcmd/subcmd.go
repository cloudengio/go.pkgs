// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package subcmd provides a multi-level command facility of the following form:
//
//   Usage of <tool>
//     <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
//       <sub-command-2-1> <flags for sub-command-2-1> <args for sub-comand-2-1>
//       ...
//       <sub-command-2-2> <flags for sub-command-2-2> <args for sub-comand-2-2>
//     ...
//     <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>
//
// The primary motivation for this package was to avoid the need to use global
// variables to store flag values packages. Such global variables quickly
// become a maintenance problem as command line tools evolve and in particular
// as functions are refactored. The cloudeng.io/cmdutil/flags package provides
// a means of defining flags as fields in a struct with a struct
// tag providing the flag name, default value and usage and is used to
// represent all flags.
//
// subcmd builds on the standard flag package and mirrors its design but
// without requiring that flag.Parse or any of its global state be used.
//
// Flags are represented by a FlagSet which encapsulates an underlying
// flag.FlagSet but with flag variables provided via cloudeng.io/cmdutil/flags.
//
// The Command type associates a FlagSet with the function that implements
// that command as well as documenting the command. This 'runner' takes as an
// argument the struct used to store its flag values as well as the command
// line arguments; thus avoiding the need for global flag variables at the cost
// of a type assertion.
// A CommandSet is used to create the command hierarchy itself and finally
// the cmdset can be used to dispatch the appropriate command functions
// via cmdset.Dispatch or DispatchWithArgs.
//
//    type rangeFlags struct {
//      From int `subcmd:"from,1,start value for a range"`
//      To   int `subcmd:"to,2,end value for a range "`
//    }
//    func printRange(ctx context.Context, values interface{}, args []string) error {
//      r := values.(*rangeFlags)
//      fmt.Printf("%v..%v\n", r.From, r.To)
//      return nil
//    }
//
//    func main() {
//      ctx := context.Background()
//      fs := subcmd.NewFlagSet()
//      fs.MustRegisterFlagStruct(&rangeFlags{}, nil, nil)
//      // Subcommands are added using the subcmd.WithSubcommands option.
//      cmd := subcmd.NewCommand("ranger", fs, printRange, subcmd.WithoutArguments())
//      cmd.Document("print an integer range")
//      cmdSet := subcmd.NewCommandSet(cmd)
//      if err := cmdSet.Dispatch(ctx); err != nil {
//         panic(err)
//      }
//    }
//
// Note that this package will never call flag.Parse and will not associate
// any flags with flag.CommandLine.
package subcmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/flags"
)

// FlagSet represents the name, description and flag values for a command.
type FlagSet struct {
	flagSet    *flag.FlagSet
	flagValues interface{}
	sm         *flags.SetMap
}

// NewFlagSet returns a new instance of FlagSet.
func NewFlagSet() *FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.Usage = func() {}
	fs.SetOutput(ioutil.Discard)
	return &FlagSet{flagSet: fs}
}

// RegisterFlagStruct registers a struct, using flags.RegisterFlagsInStructWithSetMap.
// The struct tag must be 'subcomd'. The returned SetMap can be queried by the
// IsSet method.
func (cf *FlagSet) RegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error {
	sm, err := flags.RegisterFlagsInStructWithSetMap(cf.flagSet, "subcmd", flagValues, valueDefaults, usageDefaults)
	cf.flagValues = flagValues
	cf.sm = sm
	return err
}

// MustRegisterFlagStruct is like RegisterFlagStruct except that it panics
// on encountering an error. Its use is encouraged over RegisterFlagStruct from
// within init functions.
func (cf *FlagSet) MustRegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) {
	err := cf.RegisterFlagStruct(flagValues, valueDefaults, usageDefaults)
	if err != nil {
		panic(err)
	}
}

// IsSet returns true if the supplied flag variable's value has been
// set, either via a string literal in the struct or via the valueDefaults
// argument to RegisterFlagStruct.
func (cf *FlagSet) IsSet(field interface{}) (string, bool) {
	return cf.sm.IsSet(field)
}

// Runner is the type of the function to be called to run a particular command.
type Runner func(ctx context.Context, flagValues interface{}, args []string) error

// Command represents a single command.
type Command struct {
	name        string
	description string
	arguments   string
	runner      Runner
	flags       *FlagSet
	opts        options
}

// NewCommand returns a new instance of Command.
func NewCommand(name string, flags *FlagSet, runner Runner, options ...CommandOption) *Command {
	cmd := &Command{
		name:   name,
		runner: runner,
		flags:  flags,
	}
	for _, fn := range options {
		fn(&cmd.opts)
	}
	return cmd
}

func (cmd *Command) Document(description string, arguments ...string) {
	cmd.description = description
	cmd.arguments = strings.Join(arguments, " ")
}

func namesAndDefault(name string, fs *flag.FlagSet) string {
	summary := []string{}
	fs.VisitAll(func(fl *flag.Flag) {
		summary = append(summary, "--"+fl.Name+"="+fl.DefValue)
	})
	return name + " [" + strings.Join(summary, " ") + "]"
}

// Usage returns a string containing a 'usage' message for the command. It
// includes a summary of the command (including a list of any sub commands)
// its flags and arguments and the flag defaults.
func (cmd *Command) Usage() string {
	out := &strings.Builder{}
	fmt.Fprintf(out, "Usage of command %v", cmd.name)
	if len(cmd.description) > 0 {
		fmt.Fprintf(out, ": %v", cmd.description)
	}
	out.WriteString("\n")
	fs := cmd.flags.flagSet
	cl := namesAndDefault(cmd.name, fs)
	out.WriteString(cl)
	if sc := cmd.opts.subcmds; sc != nil {
		fmt.Fprintf(out, " %v ...", strings.Join(sc.Commands(), "|"))
	} else if args := cmd.arguments; len(args) > 0 {
		if len(cl) > 0 {
			out.WriteString(" ")
		}
		out.WriteString(args)
	}
	out.WriteString("\n")
	orig := cmd.flags.flagSet.Output()
	defer cmd.flags.flagSet.SetOutput(orig)
	cmd.flags.flagSet.SetOutput(out)
	cmd.flags.flagSet.PrintDefaults()
	return out.String()
}

// CommandSet represents a set of commands that are peers to each other,
// that is, the command line must specificy one of them.
type CommandSet struct {
	cmds []*Command
	out  io.Writer
}

// CommandOption represents an option controlling the handling of a given
// command.
type CommandOption func(*options)

type options struct {
	withoutArgs       bool
	optionalSingleArg bool
	exactArgs         bool
	numArgs           int
	subcmds           *CommandSet
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

// WithSubCommands associates a set of commands that are subordinate to the
// current one. Once all of the flags for the current command processed the
// first argument must be one of these commands.
func WithSubCommands(cmds *CommandSet) CommandOption {
	return func(o *options) {
		*o = options{}
		o.subcmds = cmds
	}
}

// First creates the first command.
func NewCommandSet(cmds ...*Command) *CommandSet {
	return &CommandSet{out: os.Stderr, cmds: cmds}
}

// Defaults returns the value of Defaults for each command in commands.
func (cmds *CommandSet) defaults() string {
	out := &strings.Builder{}
	for i, cmd := range cmds.cmds {
		out.WriteString(cmd.Usage())
		if i < len(cmds.cmds)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// Usage returns a function that can be assigned to flag.Usage.
func (cmds *CommandSet) Usage(name string) string {
	return fmt.Sprintf("Usage of %v\n\n%s", name, cmds.defaults())
}

// Commands returns the list of available commands.
func (cmds *CommandSet) Commands() []string {
	out := make([]string, len(cmds.cmds))
	for i, cmd := range cmds.cmds {
		out[i] = cmd.name
	}
	return out
}

// Dispatch will dispatch the appropriate sub command or return an error.
func (cmds *CommandSet) Dispatch(ctx context.Context) error {
	return cmds.DispatchWithArgs(ctx, filepath.Base(os.Args[0]), os.Args[1:]...)
}

// MustDispatch will dispatch the appropriate sub command or exit.
func (cmds *CommandSet) MustDispatch(ctx context.Context) {
	err := cmds.DispatchWithArgs(ctx, filepath.Base(os.Args[0]), os.Args[1:]...)
	if err != nil {
		cmdutil.Exit("%v", err)
	}
}

// SetOutput is like flag.FlagSet.SetOutput.
func (cmds *CommandSet) SetOutput(out io.Writer) {
	cmds.out = out
}

// Output is like flag.FlagSet.Output.
func (cmds *CommandSet) Output() io.Writer {
	return cmds.out
}

// Dispatch determines which top level command has been requested, if any,
// parses the command line appropriately and then runs its associated function.
func (cmds *CommandSet) DispatchWithArgs(ctx context.Context, usage string, args ...string) error {
	if len(args) == 0 {
		fmt.Fprintln(cmds.out, cmds.Usage(usage))
		return fmt.Errorf("missing top level command: available commands are: %v", strings.Join(cmds.Commands(), ", "))
	}
	requested := args[0]
	switch requested {
	case "help", "-help", "--help", "-h", "--h":
		fmt.Fprintln(cmds.out, cmds.Usage(usage))
		return flag.ErrHelp
	}
	for _, cmd := range cmds.cmds {
		fs := cmd.flags.flagSet
		if requested == cmd.name {
			if cmd.runner == nil {
				return fmt.Errorf("no runner registered for %v", requested)
			}
			args := args[1:]
			if err := fs.Parse(args); err != nil {
				if err == flag.ErrHelp {
					fmt.Println(cmd.Usage())
					return err
				}
				return fmt.Errorf("%v: failed to parse flags: %v", cmd.name, err)
			}
			args = fs.Args()
			switch {
			case cmd.opts.withoutArgs:
				if len(args) > 0 {
					return fmt.Errorf("%v: does not accept any arguments", cmd.name)
				}
			case cmd.opts.optionalSingleArg:
				if len(args) > 1 {
					return fmt.Errorf("%v: accepts at most one argument", cmd.name)
				}
			case cmd.opts.exactArgs:
				if len(args) != cmd.opts.numArgs {
					return fmt.Errorf("%v: accepts exactly %v arguments", cmd.name, cmd.opts.numArgs)
				}
			case cmd.opts.subcmds != nil:
				return cmd.opts.subcmds.DispatchWithArgs(ctx, usage, args...)
			}
			return cmd.runner(ctx, cmd.flags.flagValues, args)
		}
	}
	return fmt.Errorf("%v is not one of the supported commands: %v", requested, strings.Join(cmds.Commands(), ", "))
}
