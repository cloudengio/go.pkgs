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

### Func ExpandEnv
```go
func ExpandEnv(e string) string
```
ExpandEnv is like os.ExpandEnv but supports 'pseudo' environment variables
that have OS specific handling as follows:

$USERHOME is replaced by $HOME on unix-like sytems and
$HOMEDRIVE:\\$HOMEPATH on windows. On windows, / are replaced with \.

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

Default values may contain shell variables as per os.ExpandEnv. So
$HOME/.configdir may be used for example.

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
### Type ColonRangeSpec
```go
type ColonRangeSpec struct {
	RangeSpec
}
```
ColonRangeSpec is like RangeSpec except that : is the separator.

### Methods

```go
func (crs *ColonRangeSpec) Set(v string) error
```
Set implements flag.Value.


```go
func (crs *ColonRangeSpec) String() string
```
String implements flag.Value.




### Type ColonRangeSpecs
```go
type ColonRangeSpecs []ColonRangeSpec
```
ColonRangeSpecs represents comma separated list of ColonRangeSpec's.

### Methods

```go
func (crs *ColonRangeSpecs) Set(val string) error
```
Set implements flag.Value.


```go
func (crs *ColonRangeSpecs) String() string
```
String implements flag.Value.




### Type Commas
```go
type Commas struct {
	Values   []string
	Validate func(string) error
}
```
Commas represents the values for flags that contain comma separated values.
The optional validate function is applied to each sub value separately.

### Methods

```go
func (c *Commas) Set(v string) error
```
Set implements flag.Value.


```go
func (c *Commas) String() string
```
String inplements flag.Value.




### Type ErrInvalidRange
```go
type ErrInvalidRange struct {
	// contains filtered or unexported fields
}
```
ErrInvalidRange represents the error generated for an invalid range. Use
errors.Is to test for it.

### Methods

```go
func (ire *ErrInvalidRange) Error() string
```
Error implements error.


```go
func (ire ErrInvalidRange) Is(target error) bool
```
Is implements errors.Is.




### Type ErrMap
```go
type ErrMap struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (me *ErrMap) Error() string
```
Error implements error.


```go
func (me ErrMap) Is(target error) bool
```
Is implements errors.Is.




### Type IntRangeSpec
```go
type IntRangeSpec struct {
	From, To      int
	RelativeToEnd bool
	ExtendsToEnd  bool
}
```
IntRangeSpec represents ranges whose values must be integers.

### Methods

```go
func (ir *IntRangeSpec) Set(val string) error
```
Set implements flag.Value.


```go
func (ir *IntRangeSpec) String() string
```
String implements flag.Value.




### Type IntRangeSpecs
```go
type IntRangeSpecs []IntRangeSpec
```
IntRangeSpecs represents a comma separated list of IntRangeSpec's.

### Methods

```go
func (irs *IntRangeSpecs) Set(val string) error
```
Set implements flag.Value.


```go
func (irs *IntRangeSpecs) String() string
```
String implements flag.Value.




### Type Map
```go
type Map struct {
	// contains filtered or unexported fields
}
```
Map represents a mapping of strings to values that implements flag.Value and
can be used for command line flag values. It must be appropriately
initialized with name, value pairs and a default value using its Register
and Default methods.

### Methods

```go
func (ef Map) Default(val interface{}) Map
```


```go
func (ef *Map) Get() interface{}
```
Value implements flag.Getter.


```go
func (ef Map) Register(name string, val interface{}) Map
```


```go
func (ef *Map) Set(v string) error
```
Set implements flag.Value.


```go
func (ef *Map) String() string
```
String implements flag.Value.




### Type OneOf
```go
type OneOf string
```
OneOf represents a string that can take only one of a fixed set of values.

### Methods

```go
func (ef OneOf) Validate(value string, values ...string) error
```
Validate ensures that the instance of OneOf has one of the specified set
values.




### Type RangeSpec
```go
type RangeSpec struct {
	From, To      string
	RelativeToEnd bool
	ExtendsToEnd  bool
}
```
RangeSpec represents a specification for a 'range' such as that used to
specify pages to be printed or table columns to be accessed. It implements
flag.Value.

Each range is of the general form:

    <from>[-<to>] | -<from>[-<to>|-] | <from>-

which allows for the following:

    <from>        : a single item
    <from>-<to>   : a range of one or more items
    -<from>       : a single item, relative to the end
    -<from>-<to>  : a range, whose start and end are indexed relative the end
    -<from>-      : a range, relative to the end that extends to the end
    <from>-       : a range that extends to the end

Note that the interpretation of these ranges is left to users of this type.
For example, intepreting these values as pages in a document could lead to
the following:

     3      : page 3
    2-4     : pages 2 through 4
    4-2     : pages 4 through 2
     -2     : second to last page
    -4-2    : fourth from last to second from last
    -2-4    : second from last to fourth from last
    -2-     : second to last and all following pages
    2-      : page 2 and all following pages.

### Methods

```go
func (rs *RangeSpec) Set(v string) error
```
Set implements flag.Value.


```go
func (rs RangeSpec) String() string
```
String implements flag.Value.




### Type RangeSpecs
```go
type RangeSpecs []RangeSpec
```
RangeSpecs represents comma separated list of RangeSpec's.

### Methods

```go
func (rs *RangeSpecs) Set(val string) error
```
Set implements flag.Value.


```go
func (rs *RangeSpecs) String() string
```
String implements flag.Value.




### Type Repeating
```go
type Repeating struct {
	Values   []string
	Validate func(string) error
}
```
Repeating represents the values from multiple instances of the same command
line argument.

### Methods

```go
func (r *Repeating) Get() interface{}
```
Get inplements flag.Getter.


```go
func (r *Repeating) Set(v string) error
```
Set inplements flag.Value.


```go
func (r *Repeating) String() string
```
String inplements flag.Value.




### Type SetMap
```go
type SetMap struct {
	// contains filtered or unexported fields
}
```
SetMaps represents flag variables, indexed by their address, whose value has
someone been set.

### Functions

```go
func RegisterFlagsInStructWithSetMap(fs *flag.FlagSet, tag string, structWithFlags interface{}, valueDefaults map[string]interface{}, usageDefaults map[string]string) (*SetMap, error)
```
RegisterFlagsInStructWithSetMap is like RegisterFlagsInStruct but returns a
SetMap which can be used to determine which flag variables have been
initialized either with a literal in the struct tag or via the valueDefaults
argument.



### Methods

```go
func (sm *SetMap) IsSet(field interface{}) (string, bool)
```
IsSet returns true if the supplied flag variable's value has been set,
either via a string literal in the struct or via the valueDefaults argument
to RegisterFlagsInStructWithSetMap.






## Examples
### [ExampleRegisterFlagsInStruct](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-RegisterFlagsInStruct)

### [ExampleMap](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-Map)

### [ExampleRangeSpecs](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-RangeSpecs)




