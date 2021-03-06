# Package [cloudeng.io/cmdutil/subcmd](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/subcmd)](https://goreportcard.com/report/cloudeng.io/cmdutil/subcmd)

```go
import cloudeng.io/cmdutil/subcmd
```

Package subcmd provides a multi-level command facility of the following
form:

    Usage of <tool>
      <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
        <sub-command-2-1> <flags for sub-command-2-1> <args for sub-comand-2-1>
        ...
        <sub-command-2-2> <flags for sub-command-2-2> <args for sub-comand-2-2>
      ...
      <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>

The primary motivation for this package was to avoid the need to use global
variables to store flag values packages. Such global variables quickly
become a maintenance problem as command line tools evolve and in particular
as functions are refactored. The cloudeng.io/cmdutil/flags package provides
a means of defining flags as fields in a struct with a struct tag providing
the flag name, default value and usage and is used to represent all flags.

subcmd builds on the standard flag package and mirrors its design but
without requiring that flag.Parse or any of its global state be used.

Flags are represented by a FlagSet which encapsulates an underlying
flag.FlagSet but with flag variables provided via cloudeng.io/cmdutil/flags.

The Command type associates a FlagSet with the function that implements that
command as well as documenting the command. This 'runner' takes as an
argument the struct used to store its flag values as well as the command
line arguments; thus avoiding the need for global flag variables at the cost
of a type assertion. A CommandSet is used to create the command hierarchy
itself and finally the cmdset can be used to dispatch the appropriate
command functions via cmdset.Dispatch or DispatchWithArgs.

    type rangeFlags struct {
      From int `subcmd:"from,1,start value for a range"`
      To   int `subcmd:"to,2,end value for a range "`
    }
    func printRange(ctx context.Context, values interface{}, args []string) error {
      r := values.(*rangeFlags)
      fmt.Printf("%v..%v\n", r.From, r.To)
      return nil
    }

    func main() {
      ctx := context.Background()
      fs := subcmd.NewFlagSet()
      fs.MustRegisterFlagStruct(&rangeFlags{}, nil, nil)
      // Subcommands are added using the subcmd.WithSubcommands option.
      cmd := subcmd.NewCommand("ranger", fs, printRange, subcmd.WithoutArguments())
      cmd.Document("print an integer range")
      cmdSet := subcmd.NewCommandSet(cmd)
      cmdSet.MustDispatch(ctx)
    }

In addition it is possible to register 'global' flags that may be specified
before any sub commands on invocation and also to wrap calls to any
subcommand's runner function. The former is useful for setting common flags
and the latter for acting on those flags and/or implementing common
functionality such as profiling or initializing logging etc.

Note that this package will never call flag.Parse and will not associate any
flags with flag.CommandLine.

## Types
### Type Command
```go
type Command struct {
	// contains filtered or unexported fields
}
```
Command represents a single command.

### Functions

```go
func NewCommand(name string, flags *FlagSet, runner Runner, options ...CommandOption) *Command
```
NewCommand returns a new instance of Command.


```go
func NewCommandLevel(name string, subcmds *CommandSet) *Command
```
NewCommandLevel returns a new instance of Command with subcommands.



### Methods

```go
func (cmd *Command) Document(description string, arguments ...string)
```
Document adds a description of the command and optionally descriptions of
its arguments.


```go
func (cmd *Command) Usage() string
```
Usage returns a string containing a 'usage' message for the command. It
includes a summary of the command (including a list of any sub commands) its
flags and arguments and the flag defaults.




### Type CommandOption
```go
type CommandOption func(*options)
```
CommandOption represents an option controlling the handling of a given
command.

### Functions

```go
func AtLeastNArguments(n int) CommandOption
```
AtLeastNArguments specifies that the command takes at least N arguments.


```go
func ExactlyNumArguments(n int) CommandOption
```
ExactlyNumArguments specifies that the command takes exactly the specified
number of arguments.


```go
func OptionalSingleArgument() CommandOption
```
OptionalSingleArg specifies that the command takes an optional single
argument.


```go
func WithoutArguments() CommandOption
```
WithoutArguments specifies that the command takes no arguments.




### Type CommandSet
```go
type CommandSet struct {
	// contains filtered or unexported fields
}
```
CommandSet represents a set of commands that are peers to each other, that
is, the command line must specificy one of them.

### Functions

```go
func NewCommandSet(cmds ...*Command) *CommandSet
```
NewCommandSet creates a new command set.



### Methods

```go
func (cmds *CommandSet) Commands() []string
```
Commands returns the list of available commands.


```go
func (cmds *CommandSet) Defaults(name string) string
```
Defaults returns the usage message and flag defaults.


```go
func (cmds *CommandSet) Dispatch(ctx context.Context) error
```
Dispatch will dispatch the appropriate sub command or return an error.


```go
func (cmds *CommandSet) DispatchWithArgs(ctx context.Context, usage string, args ...string) error
```
Dispatch determines which top level command has been requested, if any,
parses the command line appropriately and then runs its associated function.


```go
func (cmds *CommandSet) Document(doc string)
```
Document adds a description for the command set.


```go
func (cmds *CommandSet) MustDispatch(ctx context.Context)
```
MustDispatch will dispatch the appropriate sub command or exit.


```go
func (cmds *CommandSet) Output() io.Writer
```
Output is like flag.FlagSet.Output.


```go
func (cmds *CommandSet) SetOutput(out io.Writer)
```
SetOutput is like flag.FlagSet.SetOutput.


```go
func (cmds *CommandSet) Summary() string
```
Summary returns a summary of the command set that includes its top level
documentation and a list of its sub-commands.


```go
func (cmds *CommandSet) Usage(name string) string
```
Usage returns the usage message for the command set.


```go
func (cmds *CommandSet) WithGlobalFlags(global *FlagSet)
```
WithGlobalFlags adds top-level/global flags that apply to all commands. They
must be specified before a subcommand, ie: command <global-flags>*
sub-command <sub-command-pflags>* args


```go
func (cmds *CommandSet) WithMain(m Main)
```
WithMain arranges for Main to be called by Dispatch to wrap the call to the
requested RunnerFunc.




### Type FlagSet
```go
type FlagSet struct {
	// contains filtered or unexported fields
}
```
FlagSet represents the name, description and flag values for a command.

### Functions

```go
func GlobalFlagSet() *FlagSet
```
GlobalFlagSet creates a new FlagSet that is to be used for global flags.


```go
func MustRegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) *FlagSet
```
MustRegisterFlagStruct is like RegisterFlagStruct except that it panics on
encountering an error. Its use is encouraged over RegisterFlagStruct from
within init functions.


```go
func NewFlagSet() *FlagSet
```
NewFlagSet returns a new instance of FlagSet.


```go
func RegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) (*FlagSet, error)
```
RegisterFlagStruct creates a new FlagSet and calls RegisterFlagStruct on it.



### Methods

```go
func (cf *FlagSet) IsSet(field interface{}) (string, bool)
```
IsSet returns true if the supplied flag variable's value has been set,
either via a string literal in the struct or via the valueDefaults argument
to RegisterFlagStruct.


```go
func (cf *FlagSet) MustRegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string)
```
MustRegisterFlagStruct is like RegisterFlagStruct except that it panics on
encountering an error. Its use is encouraged over RegisterFlagStruct from
within init functions.


```go
func (cf *FlagSet) RegisterFlagStruct(flagValues interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error
```
RegisterFlagStruct registers a struct, using
flags.RegisterFlagsInStructWithSetMap. The struct tag must be 'subcomd'. The
returned SetMap can be queried by the IsSet method.




### Type Main
```go
type Main func(ctx context.Context, cmdRunner func() error) error
```
Main is the type of the function that can be used to intercept a call to a
Runner.


### Type Runner
```go
type Runner func(ctx context.Context, flagValues interface{}, args []string) error
```
Runner is the type of the function to be called to run a particular command.




## Examples
### [ExampleCommandSet](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd?tab=doc#example-CommandSet)




