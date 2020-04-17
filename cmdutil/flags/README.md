# Package [cloudeng.io/cmdutil/flags](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/flags)](https://goreportcard.com/report/cloudeng.io/cmdutil/flags)

```go
import cloudeng.io/cmdutil/flags
```

Package flags provides support for working with flag variables, and for
managing flag variables by embedding them in structs. A field in a struct
can be annotated with a tag that is used to identify it as a variable to be
registered with a flag that contains the name of the flag, an initial
default value and the usage message. This makes it convenient to colocate
flags with related data structures and to avoid large numbers of global
variables as are often encountered with complex, multi-level command
structures.

## Functions
### Func AllSet
```go
func AllSet(args ...interface{}) bool
```
AllSet is like ExactlyOne except that it returns true if all of its
arguments are set.

### Func AtMostOneSet
```go
func AtMostOneSet(args ...interface{}) bool
```
AtMostOneSet is like ExactlyOne except that it returns true if zero or one
of its arguments are set.

### Func ExactlyOneSet
```go
func ExactlyOneSet(args ...interface{}) bool
```
ExactlyOneSet will return true if exactly one of its arguments is 'set',
where 'set' means:

    1. for strings, the length is > 0.
    2. fo slices, arrays and maps, their length is > 0.

ExactlyOneSet will panic if any of the arguments are not one of the above
types.

### Func ParseFlagTag
```go
func ParseFlagTag(t string) (name, value, usage string, err error)
```
ParseFlagTag parses the supplied string into a flag name, default literal
value and description components. It is used by
CreatenAndRegisterFlagsInStruct to parse the field tags.

The tag format is:

<name>,<default-value>,<usage>

where <name> is the name of the flag, <default-value> is an optional literal
default value for the flag and <usage> the detailed description for the
flag. <default-value> may be left empty, but <name> and <usage> must be
supplied. All fields can be quoted if they need to contain a comma.

### Func RegisterFlagsInStruct
```go
func RegisterFlagsInStruct(fs *flag.FlagSet, tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) error
```
RegisterFlagsInStruct will selectively register fields in the supplied
struct as flags of the appropriate type with the supplied flag.FlagSet.
Fields are selected if they have tag of the form
`cmdline:"name::<literal>,<usage>"` associated with them, as defined by
ParseFlagTag above. In addition to literal default values specified in the
tag it is possible to provide computed default values via the
valuesDefaults, and also defaults that will appear in the usage string for
help messages that override the actual default value. The latter is useful
for flags that have a default that is system dependent that is not
informative in the usage statement. For example --home-dir which should
default to /home/user but the usage message would more usefully say
--home-dir=$HOME. Both maps are keyed by the name of the flag, not the
field.

Embedded (anonymous) structs may be used provided that they are not
themselves tagged. For example:

type CommonFlags struct {

    A int `cmdline:"a,,use a"`
    B int `cmdline:"b,,use b"`

}

flagSet := struct{

    CommonFlags
    C bool `cmdline:"c,,use c"`

}

will result in three flags, --a, --b and --c. Note that embedding as a
pointer is not supported.



## Types
### Type Commas
```go
type Commas struct {
	Values   []string
	Validate func(string) error
}
```
Commas represents the values for flags that contain comma separated values.
The optional validate function is applied to each sub value separately.

### Type Repeating
```go
type Repeating struct {
	Values   []string
	Validate func(string) error
}
```
Repeating represents the values from multiple instances of the same command
line argument.



## Examples

### [ExampleRegisterFlagsInStruct](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-RegisterFlagsInStruct)



