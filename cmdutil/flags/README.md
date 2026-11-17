# Package [cloudeng.io/cmdutil/flags](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc)

```go
import cloudeng.io/cmdutil/flags
```

Package flags provides support for working with flag variables,
and for managing flag variables by embedding them in structs. A field
in a struct can be annotated with a tag that is used to identify it as a
variable to be registered with a flag that contains the name of the flag,
an initial default value and the usage message. This makes it convenient to
colocate flags with related data structures and to avoid large numbers of
global variables as are often encountered with complex, multi-level command
structures.

## Constants
### IntervalSeparator
```go
const IntervalSeparator = ".."
```
IntervalSeparator is the default separator between the two ends of an
Interval, see Interval.SetSeparator.

### ListSeparator
```go
const ListSeparator = ","
```
ListSeparator separates the intervals of an Intervals.



## Variables
### ErrInvalidValue
```go
ErrInvalidValue = errors.New("invalid value")

```
ErrInvalidValue is wrapped by the error returned for any value that cannot
be parsed by Bound, Interval or Intervals, use errors.Is to test for it.
The error also wraps the underlying cause, if there is one.



## Functions
### Func AllSet
```go
func AllSet(args ...any) bool
```
AllSet is like ExactlyOne except that it returns true if all of its
arguments are set.

### Func AtMostOneSet
```go
func AtMostOneSet(args ...any) bool
```
AtMostOneSet is like ExactlyOne except that it returns true if zero or one
of its arguments are set.

### Func ExactlyOneSet
```go
func ExactlyOneSet(args ...any) bool
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

On Windows $HOME and $PATH are replaced by and $HOMEDRIVE:\\$HOMEPATH and
$Path respectively. On Windows /'s are replaced with \'s.

### Func ParseFlagTag
```go
func ParseFlagTag(t string) (name, value, usage string, err error)
```
ParseFlagTag parses the supplied string into a flag name,
default literal value and description components. It is used by
CreatenAndRegisterFlagsInStruct to parse the field tags.

The tag format is:

<name>,<default-value>,<usage>

where <name> is the name of the flag, <default-value> is an optional literal
default value for the flag and <usage> the detailed description for the
flag. <default-value> may be left empty, but <name> and <usage> must be
supplied. All fields can be quoted if they need to contain a comma.

Default values may contain shell variables as per flags.ExpandEnv.
So $HOME/.configdir may be used on both UNIX and Windows for example.

A struct tag can only hold a literal. If <default-value> is empty and the
flag variable has a 'DefaultValues() string' method, as a flag.Value can,
then RegisterFlagsInStruct uses its result as the default instead.
A default, from either source, is passed to the flag variable's
'SetDefault(string) error' method if it has one, so that it can tell a
default from a value that was set by the user, and to Set otherwise.

### Func RegisterAndParse
```go
func RegisterAndParse[T any](tag string, values *T) error
```
RegisterAndParse registers the flags in the supplied struct and parses the
command line arguments. It uses RegisterFlagsInStruct with the supplied
tag and no default option values. The following struct can be parsed with
RegisterAndParse("flags", &myflags{}):

    type myflags struct {
      Verbose bool `flags:"verbose,false,enable verbose logging"`
    }

It returns an error if there is a problem registering the flags or parsing
the command line arguments.

### Func RegisterAndParseMust
```go
func RegisterAndParseMust[T any](tag string, values *T)
```
RegisterAndParseMust is like RegisterAndParse but panics if there is an
error.

### Func RegisterFlagsInStruct
```go
func RegisterFlagsInStruct(fs *flag.FlagSet, tag string, structWithFlags any, valueDefaults map[string]any, usageDefaults map[string]string) error
```
RegisterFlagsInStruct will selectively register fields in the
supplied struct as flags of the appropriate type with the supplied
flag.FlagSet. Fields are selected if they have tag of the form
`cmdline:"name::<literal>,<usage>"` associated with them, as defined
by ParseFlagTag above. In addition to literal default values specified
in the tag it is possible to provide computed default values via the
valuesDefaults, and also defaults that will appear in the usage string
for help messages that override the actual default value. The latter is
useful for flags that have a default that is system dependent that is not
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

### Func WithDefault
```go
func WithDefault[T comparable](a, b T) T
```
WithDefault returns a value if it is not the zero value for its type,
otherwise it returns the second value. This is useful for command line flags
where a default value should be used if the user does not provide one.



## Types
### Type Bound
```go
type Bound[T any, C Codec[T]] struct {
	// contains filtered or unexported fields
}
```
Bound is a flag.Value for a single value of type T that is compared with
others, such as --since or --max-size. It names a span, see Edge, and so a
time or date can be asked about as a whole. The zero value is unset.

A Bound is encoded to and from text, such as for JSON and YAML, as the text
it was set from.

### Methods

```go
func (b *Bound[T, C]) After(v T) bool
```
After returns true if the whole of the span that the bound names is after v,
ie. if v is before its Lower edge. It is false if the bound is not set.


```go
func (b *Bound[T, C]) Before(v T) bool
```
Before returns true if the whole of the span that the bound names is before
v, ie. if v is at or after its Upper edge. It is false if the bound is not
set.


```go
func (b *Bound[T, C]) Contains(v T) bool
```
Contains returns true if v is within the span that the bound names. It is
false if the bound is not set.


```go
func (b *Bound[T, C]) Get() any
```
Get implements flag.Getter, returning the lower edge.


```go
func (b *Bound[T, C]) IsDefault() bool
```
IsDefault returns true if the value has not been set.


```go
func (b *Bound[T, C]) Lower() T
```
Lower returns the first value that the bound names.


```go
func (b Bound[T, C]) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler.


```go
func (b *Bound[T, C]) Set(s string) error
```
Set implements flag.Value.


```go
func (b *Bound[T, C]) String() string
```
String implements flag.Value, returning the value as it was set.


```go
func (b *Bound[T, C]) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler.


```go
func (b *Bound[T, C]) Upper() T
```
Upper returns the value just after the last that the bound names.




### Type ByteSize
```go
type ByteSize struct {
	// contains filtered or unexported fields
}
```
ByteSize represents a quantity of bytes that can be used as a flag.Value,
accepting the same notations as cmdtypes.ByteSize: a bare number of bytes,
or a number with a binary (KiB, MiB, GiB, TiB) or decimal (KB, MB, GB,
TB) suffix, with an optional space and in any case. It is parsed by
cmdtypes.ByteSize, so a size given on the command line is read the same way
as one read from JSON or YAML.

### Methods

```go
func (b *ByteSize) ByteSize() cmdtypes.ByteSize
```
ByteSize returns the value as a cmdtypes.ByteSize, for use where the same
value is also read from or written to JSON or YAML.


```go
func (b *ByteSize) Bytes() int64
```
Bytes returns the number of bytes represented by the flag.


```go
func (b *ByteSize) Get() any
```
Get implements flag.Getter, returning the number of bytes.


```go
func (b *ByteSize) IsDefault() bool
```
IsDefault returns true if the value has not been set.


```go
func (b *ByteSize) Set(v string) error
```
Set implements flag.Value.


```go
func (b *ByteSize) String() string
```
String implements flag.Value. It returns the size as supplied, so that a
default retains the units it was written in.




### Type ByteSizeBound
```go
type ByteSizeBound = Bound[cmdtypes.ByteSize, ByteSizeCodec]
```
ByteSizeBound is a Bound of a number of bytes.


### Type ByteSizeCodec
```go
type ByteSizeCodec struct{}
```
ByteSizeCodec is the Codec for cmdtypes.ByteSize, accepting the same
notations as it.

### Methods

```go
func (ByteSizeCodec) Compare(a, b cmdtypes.ByteSize) int
```
Compare implements Codec.


```go
func (ByteSizeCodec) Parse(s string, edge Edge) (cmdtypes.ByteSize, error)
```
Parse implements Codec.




### Type ByteSizeInterval
```go
type ByteSizeInterval = Interval[cmdtypes.ByteSize, ByteSizeCodec]
```
ByteSizeInterval is an Interval of numbers of bytes.


### Type ByteSizeIntervals
```go
type ByteSizeIntervals = Intervals[cmdtypes.ByteSize, ByteSizeCodec]
```
ByteSizeIntervals is an Intervals of numbers of bytes.


### Type Codec
```go
type Codec[T any] interface {
	// Parse parses s, returning the edge of the span that it names.
	Parse(s string, edge Edge) (T, error)
	// Compare returns -1, 0 or +1 as a is before, equal to or after b.
	Compare(a, b T) int
}
```
Codec defines how a type is parsed and compared by Bound and Interval.
It is a type parameter, rather than a value, so that a Bound or Interval
has a usable zero value and so can be a field of a struct whose flags are
registered by RegisterFlagsInStruct. Implementations are therefore empty
struct types, such as TimeCodec.


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




### Type DurationBound
```go
type DurationBound = Bound[time.Duration, DurationCodec]
```
DurationBound is a Bound of a duration.


### Type DurationCodec
```go
type DurationCodec struct{}
```
DurationCodec is the Codec for time.Duration, accepting the notations of
cmdtypes.ParseDuration, with a leading '+' or '-' allowed.

### Methods

```go
func (DurationCodec) Compare(a, b time.Duration) int
```
Compare implements Codec.


```go
func (DurationCodec) Parse(s string, edge Edge) (time.Duration, error)
```
Parse implements Codec.




### Type DurationInterval
```go
type DurationInterval = Interval[time.Duration, DurationCodec]
```
DurationInterval is an Interval of durations.


### Type DurationIntervals
```go
type DurationIntervals = Intervals[time.Duration, DurationCodec]
```
DurationIntervals is an Intervals of durations.


### Type Edge
```go
type Edge int
```
Edge says which end of the span that a value names is wanted. A value names
all of the span to its precision, so the date 2024-03-05 is the whole
of that day: its Lower edge is the start of the day and its Upper edge
the start of the next. Values with no coarser precision, such as an int,
have a span of one, so the Upper edge of 5 is 6.

### Constants
### Lower, Upper
```go
const (
	Lower Edge = iota
	Upper
)
```




### Type Enum
```go
type Enum[T EnumType[T]] struct {
	Value T
}
```
Enum is a generic flag.Value wrapper that supports enumerated types.

### Methods

```go
func (e Enum[T]) AllowedValues() string
```
AllowedValues returns a sorted, comma-separated list of all valid string
representations for this Enum.


```go
func (e Enum[T]) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler.


```go
func (e *Enum[T]) Set(val string) error
```
Set implements flag.Value and returns available options on invalid input.


```go
func (e Enum[T]) String() string
```
String implements flag.Value. If multiple strings map to the same value
the lexicographically smallest one is returned so that the result is stable
regardless of map iteration order.


```go
func (e *Enum[T]) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler.




### Type EnumType
```go
type EnumType[T any] interface {
	comparable
	EnumValues[T]
}
```
EnumType combines comparable underlying types with EnumValues.


### Type EnumValues
```go
type EnumValues[T any] interface {
	EnumValues() map[string]T
}
```
EnumValues must be implemented by any type that is to be used with Enum.


### Type IntBound
```go
type IntBound = Bound[int, IntCodec]
```
IntBound is a Bound of an int.


### Type IntCodec
```go
type IntCodec struct{}
```
IntCodec is the Codec for int. It also supports values relative to the end
of an Interval, see Interval.Resolve.

### Methods

```go
func (IntCodec) Compare(a, b int) int
```
Compare implements Codec.


```go
func (IntCodec) Parse(s string, edge Edge) (int, error)
```
Parse implements Codec.




### Type IntInterval
```go
type IntInterval = Interval[int, IntCodec]
```
IntInterval is an Interval of ints.


### Type IntIntervals
```go
type IntIntervals = Intervals[int, IntCodec]
```
IntIntervals is an Intervals of ints.


### Type Interval
```go
type Interval[T any, C Codec[T]] struct {
	// contains filtered or unexported fields
}
```
Interval is a flag.Value for a range of values of type T, [From, To),
that either end of which may be left open. The zero value is unset.

It is written <from>..<to>, <from>.. or ..<to>, or as a single value:

    a..b  : from the start of a to the end of b, so 2024-03-01..2024-03-05 includes all of the fifth
    a..   : from the start of a onwards
    ..b   : up to the end of b
    a     : all of a, so 2024-03 is the month of March and 5 is exactly 5

For types that are positions, such as IntInterval, a value may be counted
back from the end with a ^ prefix, where ^1 is the last item, eg. ^3..^1 is
the last three items, 2..^2 is all but the first two and last, and ^1 alone
is the last. Such an interval must be resolved, see Resolve, before use.

The default separator is not one that any of the values that it is used with
contain, unlike '-' which dates and negative numbers do. Use SetSeparator to
use another. An Interval that is empty or ends before it starts is an error,
wrapping ErrInvalidValue.

An Interval is encoded to and from text, such as for JSON and YAML, as the
text it was set from.

### Methods

```go
func (iv *Interval[T, C]) Contains(v T) bool
```
Contains returns true if v is within the interval. It is false if the
interval is not set or is unresolved.


```go
func (iv *Interval[T, C]) From() (T, bool)
```
From returns the start of the interval, and false if it is open.


```go
func (iv *Interval[T, C]) Get() any
```
Get implements flag.Getter, returning the interval itself.


```go
func (iv *Interval[T, C]) IsDefault() bool
```
IsDefault returns true if the value has not been set.


```go
func (iv Interval[T, C]) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler.


```go
func (iv *Interval[T, C]) Overlaps(other *Interval[T, C]) bool
```
Overlaps returns true if the intervals have any value in common. It is false
if either is nil, not set or unresolved.


```go
func (iv *Interval[T, C]) Relative() bool
```
Relative returns true if either end is relative to the end of the interval's
items, see Resolve. From and To return the offset for such an end and
Contains and Overlaps are false until the interval is resolved.


```go
func (iv *Interval[T, C]) Resolve(end T) (Interval[T, C], error)
```
Resolve returns the interval with any values that are relative to the end
replaced by absolute ones, where end is the exclusive end, such as the
number of items. It is an error if the result is empty, ends before it
starts or lies before the first item. An interval with no such values is
returned unchanged.


```go
func (iv *Interval[T, C]) Set(s string) error
```
Set implements flag.Value.


```go
func (iv *Interval[T, C]) SetSeparator(sep string)
```
SetSeparator sets the separator used by Set and UnmarshalText, which
defaults to IntervalSeparator if sep is empty. It must not appear within the
values being parsed, so '-' is a poor choice for times and negative numbers.
It is kept when the interval is later set, and should be called before the
interval is set, eg. after RegisterFlagsInStruct and before parsing the
flags. A text value is always read with the separator currently in effect. A
default value given in a struct tag is parsed when the flags are registered,
so must use the default separator.


```go
func (iv *Interval[T, C]) String() string
```
String implements flag.Value, returning the interval as it was set.


```go
func (iv *Interval[T, C]) To() (T, bool)
```
To returns the end of the interval, which is not within it, and false if it
is open.


```go
func (iv *Interval[T, C]) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler.




### Type Intervals
```go
type Intervals[T any, C Codec[T]] struct {
	// contains filtered or unexported fields
}
```
Intervals is a flag.Value for a comma separated list of Interval, such
as 1..3,7,^2.., that are all-or-nothing to set. The zero value is unset.
Setting it replaces rather than adds to any previous value.

### Methods

```go
func (l *Intervals[T, C]) Contains(v T) bool
```
Contains returns true if v is within any of the intervals.


```go
func (l *Intervals[T, C]) Get() any
```
Get implements flag.Getter, returning the list itself.


```go
func (l *Intervals[T, C]) IsDefault() bool
```
IsDefault returns true if the value has not been set.


```go
func (l *Intervals[T, C]) Items() []Interval[T, C]
```
Items returns the intervals, which must not be modified.


```go
func (l Intervals[T, C]) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler.


```go
func (l *Intervals[T, C]) Overlaps(other *Interval[T, C]) bool
```
Overlaps returns true if any of the intervals overlap other.


```go
func (l *Intervals[T, C]) Resolve(end T) (Intervals[T, C], error)
```
Resolve resolves each of the intervals, see Interval.Resolve.


```go
func (l *Intervals[T, C]) Set(s string) error
```
Set implements flag.Value.


```go
func (l *Intervals[T, C]) SetSeparator(sep string)
```
SetSeparator sets the separator used within each Interval, see
Interval.SetSeparator. It must not be ListSeparator.


```go
func (l *Intervals[T, C]) String() string
```
String implements flag.Value, returning the list as it was set.


```go
func (l *Intervals[T, C]) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler.




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




### Type Permissions
```go
type Permissions struct {
	// contains filtered or unexported fields
}
```
Permissions represents file permission bits that can be used as a
flag.Value, accepting the same notations as cmdtypes.Permissions: an octal
number ("0700", "700", "0o700", "0x1c0"), an rwx string ("rwxr-xr-x",
"-rwx------", "rwx"), or symbolic chmod notation ("u=rwx,go=",
"u=rwx,go=rx").

### Methods

```go
func (p *Permissions) FileMode() fs.FileMode
```
FileMode returns the fs.FileMode represented by the flag.


```go
func (p *Permissions) Get() any
```
Get implements flag.Getter, returning the fs.FileMode represented by the
flag.


```go
func (p *Permissions) IsDefault() bool
```
IsDefault returns true if the value has not been set.


```go
func (p *Permissions) Octal() string
```
Octal returns the 4 digit octal representation of the permissions (e.g.
"0700", "4755"), whatever the notation they were supplied in. It is the
encoding used by cmdtypes.Permissions for text, JSON and YAML, so that a
value written from a flag reads back as the same value.


```go
func (p *Permissions) Permissions() cmdtypes.Permissions
```
Permissions returns the value as a cmdtypes.Permissions, for use where the
same value is also read from or written to JSON or YAML.


```go
func (p *Permissions) Set(v string) error
```
Set implements flag.Value.


```go
func (p *Permissions) String() string
```
String implements flag.Value. It returns the permissions as supplied,
so that a default retains the notation it was written in.




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
func (r *Repeating) Get() any
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
func RegisterFlagsInStructWithSetMap(fs *flag.FlagSet, tag string, structWithFlags any, valueDefaults map[string]any, usageDefaults map[string]string) (*SetMap, error)
```
RegisterFlagsInStructWithSetMap is like RegisterFlagsInStruct but returns
a SetMap which can be used to determine which flag variables have been
initialized either with a literal in the struct tag or via the valueDefaults
argument.



### Methods

```go
func (sm *SetMap) IsSet(field any) (string, bool)
```
IsSet returns true if the supplied flag variable's value has been set,
either via a string literal in the struct or via the valueDefaults argument
to RegisterFlagsInStructWithSetMap.




### Type StringBound
```go
type StringBound = Bound[string, StringCodec]
```
StringBound is a Bound of a string.


### Type StringCodec
```go
type StringCodec struct{}
```
StringCodec is the Codec for strings, which are ordered lexically. The span
of a string s ends just after it, at s followed by a NUL.

### Methods

```go
func (StringCodec) Compare(a, b string) int
```
Compare implements Codec.


```go
func (StringCodec) Parse(s string, edge Edge) (string, error)
```
Parse implements Codec.




### Type StringInterval
```go
type StringInterval = Interval[string, StringCodec]
```
StringInterval is an Interval of strings.


### Type StringIntervals
```go
type StringIntervals = Intervals[string, StringCodec]
```
StringIntervals is an Intervals of strings.


### Type Time
```go
type Time struct {
	// contains filtered or unexported fields
}
```
Time represents a time.Time that can be used as a flag.Value. It accepts
the same formats as cmdtypes.FlexTime, ie. time.RFC3339, time.DateTime,
time.TimeOnly or time.DateOnly, and is parsed by it, so that a value given
on the command line is read the same way as one read from JSON or YAML.

### Methods

```go
func (tf *Time) FlexTime() cmdtypes.FlexTime
```
FlexTime returns the value as a cmdtypes.FlexTime, for use where the same
value is also read from or written to JSON or YAML.


```go
func (tf *Time) Get() any
```
Get implements flag.Getter, returning the time.Time represented by the flag.


```go
func (tf *Time) IsDefault() bool
```
IsSet returns true if the value has been set.


```go
func (tf *Time) Set(v string) error
```
Set implements flag.Value.


```go
func (tf *Time) String() string
```
String implements flag.Value.


```go
func (tf *Time) Time() time.Time
```
Time returns the time.Time represented by the flag.




### Type TimeBound
```go
type TimeBound = Bound[time.Time, TimeCodec]
```
TimeBound is a Bound of a time, see TimeCodec.


### Type TimeCodec
```go
type TimeCodec struct{}
```
TimeCodec is the Codec for time.Time, using cmdtypes.DefaultTimeParser: a
date and time, a date, month or year, or a time relative to now such as -7d,
see cmdtypes.TimeSpanFormats.

### Methods

```go
func (TimeCodec) Compare(a, b time.Time) int
```
Compare implements Codec.


```go
func (TimeCodec) Parse(s string, edge Edge) (time.Time, error)
```
Parse implements Codec.




### Type TimeInterval
```go
type TimeInterval = Interval[time.Time, TimeCodec]
```
TimeInterval is an Interval of times, see TimeCodec.


### Type TimeIntervals
```go
type TimeIntervals = Intervals[time.Time, TimeCodec]
```
TimeIntervals is an Intervals of times.




## Examples
### [Example_jsonEnumMarshal](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-_jsonEnumMarshal)

### [Example_jsonEnumUnmarshal](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-_jsonEnumUnmarshal)

### [Example_yamlEnumMarshal](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-_yamlEnumMarshal)

### [Example_yamlEnumUnmarshal](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-_yamlEnumUnmarshal)

### [ExampleRegisterFlagsInStruct](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-RegisterFlagsInStruct)

### [ExampleEnum](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-Enum)

### [ExampleEnum_string](https://pkg.go.dev/cloudeng.io/cmdutil/flags?tab=doc#example-Enum_string)




