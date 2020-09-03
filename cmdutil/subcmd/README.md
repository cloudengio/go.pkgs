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
      if err := cmdSet.Dispatch(ctx); err != nil {
         panic(err)
      }
    }

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



### Methods

```go
func (cmd *Command) Document(description string, arguments ...string)
```


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
func WithSubCommands(cmds *CommandSet) CommandOption
```
WithSubCommands associates a set of commands that are subordinate to the
current one. Once all of the flags for the current command processed the
first argument must be one of these commands.


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
First creates the first command.



### Methods

```go
func (cmds *CommandSet) Commands() []string
```
Commands returns the list of available commands.


```go
func (cmds *CommandSet) Dispatch(ctx context.Context) error
```


```go
func (cmds *CommandSet) DispatchWithArgs(ctx context.Context, usage string, args ...string) error
```
Dispatch determines which top level command has been requested, if any,
parses the command line appropriately and then runs its associated function.


```go
func (cmds *CommandSet) Output() io.Writer
```


```go
func (cmds *CommandSet) SetOutput(out io.Writer)
```


```go
func (cmds *CommandSet) Usage(name string) string
```
Usage returns a function that can be assigned to flag.Usage.




### Type FlagSet
```go
type FlagSet struct {
	// contains filtered or unexported fields
}
```
FlagSet represents the name, description and flag values for a command.

### Functions

```go
func NewFlagSet() *FlagSet
```
NewFlagSet returns a new instance of FlagSet.



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




### Type Runner
```go
type Runner func(ctx context.Context, flagValues interface{}, args []string) error
```
Runner is the type of the function to be called to run a particular command.




## Examples
### [ExampleCommandSet](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd?tab=doc#example-CommandSet)




