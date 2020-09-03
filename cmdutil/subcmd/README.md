# Package [cloudeng.io/cmdutil/subcmd](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/subcmd)](https://goreportcard.com/report/cloudeng.io/cmdutil/subcmd)

```go
import cloudeng.io/cmdutil/subcmd
```

Package subcmd provides a simple sub-command facility. It allows for
creating command trees of the form:

Usage of <tool>

    <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
       <sub-command-2-1> <flags for sub-command-2-1> <args for sub-comand-2-1>
       ...
       <sub-command-2-2> <flags for sub-command-2-2> <args for sub-comand-2-2>
    ...
    <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>

Creating a command consists of defining the flags with associated
descriptions and then associating the newly create Flags struct with the
function to be run to implement that command. The
flags.RegisterFlagsInStruct paradigm is used for defining flags. A
CommandSet is then created as follows:

    cmds = subcmd.First(...).Append(...).Append(...)

Once created, typically in an init function, the CommandSet may be used by
calling its Dispatch or DispatchWithArgs methods typically from the main
function.

The encapsulation of all the flag definitions within a struct and then
making that struct available to the runner function as a parameter
conveniently avoids having to define flag values at a global level.

Note that this package will never call flag.Parse and will not associate any
flags with flag.CommandLine. Commands.Usage() can be used to set flag.Usage.

## Types
### Type Command
```go
type Command struct {
	// contains filtered or unexported fields
}
```
Command represents a single command.

### Methods

```go
func (cmd Command) Usage() string
```
Usage returns a string containing a 'usage' message for the command. It
includes a summary of the command, its flags and arguments and the flag
defaults.




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
func SubCommands(cmds CommandSet) CommandOption
```
SubCommands associates a set of commands that are subordinate to the current
one. Once all flags for the current command processed the first argument
must be one of these commands.


```go
func WithoutArguments() CommandOption
```
WithoutArguments specifies that the command takes no arguments.




### Type CommandSet
```go
type CommandSet []*Command
```
CommandSet represents a set of commands that are peers to each other, that
is, the command line must specificy one of them.

### Functions

```go
func First(flags *Flags, runner Runner, options ...CommandOption) CommandSet
```
First creates the first command.



### Methods

```go
func (cmds CommandSet) Append(flags *Flags, runner Runner, options ...CommandOption) CommandSet
```
Append adds a new command.


```go
func (cmds CommandSet) Commands() []string
```
Commands returns the list of available commands.


```go
func (cmds CommandSet) Defaults() string
```
Defaults returns the value of Defaults for each command in commands.


```go
func (cmds CommandSet) Dispatch(ctx context.Context) error
```


```go
func (cmds CommandSet) DispatchWithArgs(ctx context.Context, args ...string) error
```
Dispatch determines which top level command has been requested, if any,
parses the command line appropriately and then runs its associated function.


```go
func (cmds CommandSet) Usage(name string) func()
```
Usage returns a function that can be assigned to flag.Usage.




### Type Flags
```go
type Flags struct {
	// contains filtered or unexported fields
}
```
Flags represents the name, description and flags for a command.

### Functions

```go
func NewFlags(name, description string, arguments ...string) *Flags
```
NewFlags returns a new instance of Flags. The flags are used to name and
describe the command they are associated with and optionally, the arguments
may also be given a desription.



### Methods

```go
func (cf *Flags) IsSet(field interface{}) (string, bool)
```
IsSet returns true if the supplied flag variable's value has been set,
either via a string literal in the struct or via the valueDefaults argument
to RegisterFlagStruct.


```go
func (cf *Flags) MustRegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string)
```
MustRegisterFlagStruct is like RegisterFlagStruct except that it panics on
encountering an error. Its use is encouraged over RegisterFlagStruct from
within init functions.


```go
func (cf *Flags) RegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error
```
RegisterFlagStruct registers a struct, using
RegisterFlagsInStructWithSetMap. The returned SetMap can be queried by the
IsSet method.




### Type Runner
```go
type Runner func(ctx context.Context, flagValues interface{}, args []string) error
```
Runner is the type of the function to be called to run a particular command.





