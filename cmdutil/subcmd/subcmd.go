// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package subcmd provides a multi-level command facility of the following form:
//
//	Usage of <tool>
//	  <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
//	    <sub-command-2-1> <flags for sub-command-2-1> <args for sub-comand-2-1>
//	    ...
//	    <sub-command-2-2> <flags for sub-command-2-2> <args for sub-comand-2-2>
//	  ...
//	  <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>
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
//	type rangeFlags struct {
//	  From int `subcmd:"from,1,start value for a range"`
//	  To   int `subcmd:"to,2,end value for a range "`
//	}
//
//	func printRange(ctx context.Context, values interface{}, args []string) error {
//	  r := values.(*rangeFlags)
//	  fmt.Printf("%v..%v\n", r.From, r.To)
//	  return nil
//	}
//
//
//	func main() {
//	  ctx := context.Background()
//	  fs := subcmd.MustRegisteredFlags(&rangeFlags{})
//	  // Subcommands are created using subcmd.NewCommandLevel.
//	  cmd := subcmd.NewCommand("ranger", fs, printRange, subcmd.WithoutArguments())
//	  cmd.Document("print an integer range")
//	  cmdSet := subcmd.NewCommandSet(cmd)
//	  cmdSet.MustDispatch(ctx)
//	}
//
// In addition it is possible to register 'global' flags that may be specified
// before any sub commands on invocation and also to wrap calls to any
// subcommand's runner function. The former is useful for setting common flags
// and the latter for acting on those flags and/or implementing common
// functionality such as profiling or initializing logging etc.
//
// Creating command trees with their documentation is cumbersome using the
// NewCommand and NewCommandset functions. An easier, and more readable way
// to do so is via a YAML configuration. The FromYAML function reads a yaml
// specification of a command tree, its summary documentation and argument
// specification. The returned CommandSetYAML type can then be used to
// 'decorate' the command tree with the runner functions and flag value
// instances. This is more comprehensible means of defining the command
// tree than doing so entirely via function calls. The YAML mechanism
// provides identical functionality to calling the functions directly.
//
// The YAML specification is show below and reflects the structure
// (ie. is recursive) of the command tree to be created.
//
//	name: command-name
//	summary: description of the command
//	arguments:
//	commands:
//	  - name:
//	    summary:
//	    arguments:
//	    commands:
//
// The summary is displayed when the command's usage information is displayed.
// The arguments: field is a list of the expected arguments that also defines
// the number of expected arguments.
//
//  1. If the field is missing or the list is empty then no arguments are allowed.
//
//  2. If the list contains n arguments then exactly that number of arguments
//     is expected, unless, the last argument in the list is '...' in which case
//     at least that number is
//     expected.
//
//  3. If there is a single item in the list and it is enclosed
//     in [] (in a quoted string), then 0 or 1 arguments are expected.
//
// To define a simple command line, with no sub-commands, specify only
// the name:, summary: and arguments: fields.
//
// Note that this package will never call flag.Parse and will not associate
// any flags with flag.CommandLine.
package subcmd

import (
	"bufio"
	"bytes"
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
	"cloudeng.io/text/linewrap"
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

// MustRegisteredFlagSet is a convenience function that creates a new
// FlagSet and calls RegisterFlagStruct on it. The valueDefaults and
// usageDefaults are extracted from the defaults variadic parameter.
// MustRegisteredFlagSet will panic if defaults contains inappopriate types
// for the value and usage defaults.
func MustRegisteredFlagSet(flagValues interface{}, defaults ...interface{}) *FlagSet {
	fs := NewFlagSet()
	var valueDefaults map[string]interface{}
	var usageDefaults map[string]string
	for _, def := range defaults {
		switch v := def.(type) {
		case map[string]interface{}:
			valueDefaults = v
		case map[string]string:
			usageDefaults = v
		}
	}
	fs.RegisterFlagStruct(flagValues, valueDefaults, usageDefaults)
	return fs
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
func (cf *FlagSet) MustRegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) *FlagSet {
	err := cf.RegisterFlagStruct(flagValues, valueDefaults, usageDefaults)
	if err != nil {
		panic(err)
	}
	return cf
}

// RegisterFlagStruct creates a new FlagSet and calls RegisterFlagStruct
// on it.
func RegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) (*FlagSet, error) {
	fs := NewFlagSet()
	err := fs.RegisterFlagStruct(flagValues, valueDefaults, usageDefaults)
	if err != nil {
		return nil, err
	}
	return fs, nil
}

// MustRegisterFlagStruct is like RegisterFlagStruct except that it panics
// on encountering an error. Its use is encouraged over RegisterFlagStruct from
// within init functions.
func MustRegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) *FlagSet {
	fs, err := RegisterFlagStruct(flagValues, valueDefaults, usageDefaults)
	if err != nil {
		panic(err)
	}
	return fs
}

// IsSet returns true if the supplied flag variable's value has been
// set, either via a string literal in the struct or via the valueDefaults
// argument to RegisterFlagStruct.
func (cf *FlagSet) IsSet(field interface{}) (string, bool) {
	return cf.sm.IsSet(field)
}

// Runner is the type of the function to be called to run a particular command.
type Runner func(ctx context.Context, flagValues interface{}, args []string) error

// Main is the type of the function that can be used to intercept a call to
// a Runner.
type Main func(ctx context.Context, cmdRunner func(ctx context.Context) error) error

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
	if flags == nil {
		flags = &FlagSet{}
	}
	if runner == nil {
		runner = func(ctx context.Context, values interface{}, args []string) error {
			return fmt.Errorf("no runner specified for: %v", name)
		}
	}
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

// NewCommandLevel returns a new instance of Command with subcommands.
func NewCommandLevel(name string, subcmds *CommandSet) *Command {
	cmd := &Command{
		name:  name,
		flags: NewFlagSet(),
	}
	cmd.opts.subcmds = subcmds
	return cmd
}

// Document adds a description of the command and optionally descriptions
// of its arguments.
func (cmd *Command) Document(description string, arguments ...string) {
	cmd.description = description
	cmd.arguments = strings.Join(arguments, " ")
}

func namesAndDefault(name string, fs *flag.FlagSet) string {
	summary := []string{}
	fs.VisitAll(func(fl *flag.Flag) {
		summary = append(summary, "--"+fl.Name+"="+fl.DefValue)
	})
	if len(summary) == 0 {
		return name
	}
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
	fmt.Fprintf(out, "\n%s\n", printDefaults(cmd.flags.flagSet))
	return out.String()
}

func (cmd *Command) summary() (name, desc string) {
	return cmd.name, cmd.description
}

// CommandSet represents a set of commands that are peers to each other,
// that is, the command line must specificy one of them.
type CommandSet struct {
	document   string
	global     *FlagSet
	globalMain Main
	cmd        *Command
	cmds       []*Command
	out        io.Writer
}

// CommandOption represents an option controlling the handling of a given
// command.
type CommandOption func(*options)

type options struct {
	withoutArgs       bool
	optionalSingleArg bool
	exactArgs         bool
	atLeastArgs       bool
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

// AtLeastNArguments specifies that the command takes at least N arguments.
func AtLeastNArguments(n int) CommandOption {
	return func(o *options) {
		o.atLeastArgs = true
		o.numArgs = n
	}
}

// NewCommandSet creates a new command set.
func NewCommandSet(cmds ...*Command) *CommandSet {
	return &CommandSet{out: os.Stderr, cmds: cmds}
}

// WithGlobalFlags adds top-level/global flags that apply to all
// commands. They must be specified before a subcommand, ie:
// command <global-flags>* sub-command <sub-command-pflags>* args
func (cmds *CommandSet) WithGlobalFlags(global *FlagSet) {
	cmds.global = global
}

// WithMain arranges for Main to be called by Dispatch to wrap the call
// to the requested RunnerFunc.
func (cmds *CommandSet) WithMain(m Main) {
	cmds.globalMain = m
}

func (cmds *CommandSet) TopLevel(cmd *Command) {
	cmds.cmd = cmd
}

// defaults returns the value of Defaults for each command in commands.
func (cmds *CommandSet) defaults() string {
	out := &strings.Builder{}
	out.WriteString(cmds.globalDefaults())
	for i, cmd := range cmds.cmds {
		out.WriteString(cmd.Usage())
		if i < len(cmds.cmds)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func lineWrapDefaults(input string) string {
	out := &strings.Builder{}
	sc := bufio.NewScanner(bytes.NewBufferString(input))
	block := &strings.Builder{}
	writeBlock := func() {
		if block.Len() > 0 {
			fmt.Fprintf(out, "%s\n", linewrap.Block(4, 75, block.String()))
			block.Reset()
		}
	}
	for sc.Scan() {
		l := sc.Text()
		if len(l) < 3 {
			continue
		}
		if l[:3] == "  -" {
			writeBlock()
			fmt.Fprintf(out, "%s\n", l)
			continue
		}
		fmt.Fprintf(block, "%s\n", l)
	}
	writeBlock()
	return out.String()
}

func printDefaults(fs *flag.FlagSet) string {
	out := &strings.Builder{}
	orig := fs.Output()
	fs.SetOutput(out)
	fs.PrintDefaults()
	defer fs.SetOutput(orig)
	return lineWrapDefaults(out.String())
}

func (cmds *CommandSet) globalDefaults() string {
	out := &strings.Builder{}
	if cmds.global != nil {
		fs := cmds.global.flagSet
		if cmds.cmd != nil {
			fmt.Fprintf(out, "flags:%s\n", namesAndDefault("", fs))
		} else {
			fmt.Fprintf(out, "global flags:%s\n", namesAndDefault("", fs))
		}
		fmt.Fprintf(out, "%s\n", printDefaults(fs))
	}
	return out.String()
}

// Usage returns the usage message for the command set.
func (cmds *CommandSet) Usage(name string) string {
	return fmt.Sprintf("Usage of %v\n\n%s\n", name, cmds.Summary())
}

// Defaults returns the usage message and flag defaults.
func (cmds *CommandSet) Defaults(name string) string {
	out := &strings.Builder{}
	out.WriteString(cmds.Usage(name))
	if gd := cmds.globalDefaults(); len(gd) > 0 {
		fmt.Fprintf(out, "\n%s", gd)
	}
	return out.String()
}

// Commands returns the list of available commands.
func (cmds *CommandSet) Commands() []string {
	out := make([]string, len(cmds.cmds))
	for i, cmd := range cmds.cmds {
		out[i] = cmd.name
	}
	return out
}

// Document adds a description for the command set.
func (cmds *CommandSet) Document(doc string) {
	cmds.document = doc
}

// Summary returns a summary of the command set that includes its top
// level documentation and a list of its sub-commands.
func (cmds *CommandSet) Summary() string {
	max := 0
	for _, cmd := range cmds.cmds {
		name, _ := cmd.summary()
		if l := len(name); l > max {
			max = l
		}
	}
	out := &strings.Builder{}
	if d := cmds.document; len(d) > 0 {
		fmt.Fprintf(out, "%s\n\n", linewrap.Block(1, 80, d))
	}
	for i, cmd := range cmds.cmds {
		name, desc := cmd.summary()
		fmt.Fprintf(out, " %v%v - %v", strings.Repeat(" ", max-len(name)), name, desc)
		if i < len(cmds.cmds)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
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

// GlobalFlagSet creates a new FlagSet that is to be used for global flags.
func GlobalFlagSet() *FlagSet {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {}
	flag.CommandLine.SetOutput(ioutil.Discard)
	return &FlagSet{flagSet: flag.CommandLine}
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
	if cmds.global != nil {
		fs := cmds.global.flagSet
		if err := fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				fmt.Fprintln(cmds.out, cmds.Usage(usage))
				if gd := cmds.globalDefaults(); len(gd) > 0 {
					fmt.Fprintf(cmds.out, "%s", gd)
				}
			}
			return err
		}
		args = fs.Args()
	}
	return cmds.dispatchWithArgs(ctx, usage, args)
}

func (cmds *CommandSet) mainWrapper() Main {
	wrapper := cmds.globalMain
	if wrapper == nil {
		wrapper = func(ctx context.Context, runner func(ctx context.Context) error) error {
			return runner(ctx)
		}
	}
	return wrapper
}

func (cmds *CommandSet) processHelp(usage string, args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "-help", "--help", "-h", "--h":
		fmt.Fprintln(cmds.out, cmds.Usage(usage))
		return flag.ErrHelp
	case "help":
		if cmds.cmd != nil {
			if len(args) < 2 || args[1] == cmds.cmd.name {
				fmt.Fprintln(cmds.out, cmds.cmd.Usage())
				return flag.ErrHelp
			}
			return fmt.Errorf("%v is not one of the supported commands", args[1])
		}
		if len(args) == 1 {
			fmt.Fprintln(cmds.out, cmds.Usage(usage))
			return flag.ErrHelp
		}
		for _, cmd := range cmds.cmds {
			if args[1] == cmd.name {
				fmt.Fprintln(cmds.out, cmd.Usage())
				return flag.ErrHelp
			}
		}

	}
	return nil
}

func (cmds *CommandSet) parseArgs(fs *flag.FlagSet, cmd *Command, args []string) error {
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintln(cmds.out, cmd.Usage())
			return err
		}
		return fmt.Errorf("%v: failed to parse flags: %v", cmd.name, err)
	}
	return nil
}

func (cmds *CommandSet) dispatchWithArgs(ctx context.Context, usage string, args []string) error {
	tlcmd := cmds.cmd
	if len(args) == 0 && tlcmd == nil {
		fmt.Fprintln(cmds.out, cmds.Usage(usage))
		return fmt.Errorf("no command specified")
	}

	if err := cmds.processHelp(usage, args); err != nil {
		return err
	}

	if tlcmd != nil {
		// can only be a single top-level command with no flags or arguments.
		fs := tlcmd.flags.flagSet
		if err := fs.Parse(args); err != nil {
			return err
		}
		// run top level command.
		return cmds.processChosenCmd(ctx, cmds.cmd, usage, fs.Args())
	}

	if len(args) == 0 {
		fmt.Fprintln(cmds.out, cmds.Usage(usage))
		return fmt.Errorf("no command specified")
	}

	requested := args[0]
	for _, cmd := range cmds.cmds {
		if cmd.flags == nil {
			return fmt.Errorf("no flags specified for %v", cmd.name)
		}
		fs := cmd.flags.flagSet
		if requested == cmd.name {
			if cmd.runner == nil && cmd.opts.subcmds == nil {
				return fmt.Errorf("no runner registered for %v", requested)
			}
			if err := cmds.parseArgs(fs, cmd, args[1:]); err != nil {
				return err
			}
			return cmds.processChosenCmd(ctx, cmd, usage, fs.Args())
		}
	}
	return fmt.Errorf("%v is not one of the supported commands: %v", requested, strings.Join(cmds.Commands(), ", "))
}

func (cmds *CommandSet) processChosenCmd(ctx context.Context, cmd *Command, usage string, args []string) error {
	plural := "arguments"
	if cmd.opts.numArgs == 1 {
		plural = "argument"
	}
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
			return fmt.Errorf("%v: accepts exactly %v %s", cmd.name, cmd.opts.numArgs, plural)
		}
	case cmd.opts.atLeastArgs:
		if len(args) < cmd.opts.numArgs {
			return fmt.Errorf("%v: accepts at least %v %s", cmd.name, cmd.opts.numArgs, plural)
		}
	case cmd.opts.subcmds != nil:
		if cmd.opts.subcmds.globalMain == nil {
			cmd.opts.subcmds.globalMain = cmds.globalMain
		}
		return cmd.opts.subcmds.dispatchWithArgs(ctx, usage, args)
	}
	return cmds.mainWrapper()(ctx, func(ctx context.Context) error {
		return cmd.runner(ctx, cmd.flags.flagValues, args)
	})
}
