# Package [cloudeng.io/cmdutil/subcmd](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/subcmd)](https://goreportcard.com/report/cloudeng.io/cmdutil/subcmd)

```go
import cloudeng.io/cmdutil/subcmd
```

Package subcmd provides a simple, single-level, sub-command facility. It
allows for creating single-level command trees of the form:

Usage of <tool>

    <sub-command-1> <flags for sub-command-1> <args for sub-comand-1>
    ...
    <sub-command-n> <flags for sub-command-n> <args for sub-comand-n>

## Types
### Type Command
```go
type Command struct {
	// contains filtered or unexported fields
}
```
Command represents a single top level command.


### Type CommandOption
```go
type CommandOption func(*options)
```
CommandOption represents an option controlling the handling of a given
command.

### Functions

```go
func WithoutArguments() CommandOption
```
WithoutArguments specifies that the command takes no arguments.




### Type Commands
```go
type Commands []*Command
```
Commands provides a simple implementation of structured command line
processing that supports a single level of sub-commands each with their own
set of flags.

### Methods

```go
func (c Commands) Append(flags *Flags, runner Runner, options ...CommandOption) Commands
```
Append adds a new top level command.


```go
func (c Commands) Commands() []string
```
Commands returns the list of available commands.


```go
func (c Commands) Defaults() string
```
Defaults returns the value of Defaults for each command in commands.


```go
func (c Commands) Dispatch(ctx context.Context) error
```
Dispatch determines which top level command has been requested, if any,
parses the command line appropriately and then runs its associated function.


```go
func (c Commands) Usage(name string) func()
```
Usage returns a function that can be assigned to flag.Usage.




### Type Flags
```go
type Flags struct {
	// contains filtered or unexported fields
}
```
Flags represents the name, description and flags for a single top-level
command.

### Functions

```go
func NewFlags(name, description string) *Flags
```
NewFlags returns a new instance of Flags.



### Methods

```go
func (cf *Flags) IsSet(field interface{}) (string, bool)
```
IsSet returns true if the supplied flag variable's value has been set,
either via a string literal in the struct or via the valueDefaults argument
to RegisterFlagStruct.


```go
func (cf *Flags) RegisterFlagStruct(tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error
```
RegisterFlagStruct registers a struct, using
RegisterFlagsInStructWithSetMap. The returned SetMap can be queried by the
IsSet method.




### Type Runner
```go
type Runner func(ctx context.Context, args []string) error
```
Runner is the type of the function to be called to run a single top-level
command.





