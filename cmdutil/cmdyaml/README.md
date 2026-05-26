# Package [cloudeng.io/cmdutil/cmdyaml](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc)

```go
import cloudeng.io/cmdutil/cmdyaml
```


## Constants
### Byte, KB, MB, GB, TB, KiB, MiB, GiB, TiB
```go
Byte ByteSize = 1
KB ByteSize = 1_000
MB = 1_000 * KB
GB = 1_000 * MB
TB = 1_000 * GB
KiB ByteSize = 1_024
MiB = 1_024 * KiB
GiB = 1_024 * MiB
TiB = 1_024 * GiB

```



## Functions
### Func ErrorWithSource
```go
func ErrorWithSource(spec []byte, err error) error
```
ErrorWithSource returns an error that includes the yaml source code that
was the cause of the error to help with debugging YAML errors. Note that the
errors reported for the yaml parser may be inaccurate in terms of the lines
the error is reported on. This seems to be particularly true for lists where
errors with use of tabs to indent are often reported against the previous
line rather than the offending one.

### Func ExpandEnv
```go
func ExpandEnv(cfg any, envFunc func(string) string)
```
ExpandEnv recursively expands environment variables in the fields of the
provided struct that have a 'yaml' tag. Embedded structs are also processed.
The provided envFunc is used to look up environment variable values.

### Func ParseConfig
```go
func ParseConfig(spec []byte, cfg any) error
```
ParseConfig will parse the yaml config in spec into the requested type.
It provides improved error reporting via ErrorWithSource.

### Func ParseConfigFile
```go
func ParseConfigFile(ctx context.Context, filename string, cfg any) error
```
ParseConfigFile reads a yaml config file as per ParseConfig using
file.FSReadFile to read the file. The use of FSReadFile allows for the
configuration file to be read from storage system, including from embed.FS,
instead of the local filesystem if an instance of fs.ReadFileFS is stored in
the context.

Deprecated: Use ParseConfigFiles instead.

### Func ParseConfigFileStrict
```go
func ParseConfigFileStrict(ctx context.Context, filename string, cfg any) error
```
ParseConfigFileStrict is like ParseConfigFile but reports an error if there
are unknown fields in the yaml specification.

Deprecated: Use ParseConfigFilesStrict instead.

### Func ParseConfigFiles
```go
func ParseConfigFiles(ctx context.Context, cfg any, filenames ...string) error
```
ParseConfigFiles reads and merges the YAML contents of each named file into
cfg. Files are processed in order; a field present in a later file overrides
the value set by an earlier one, while fields only in an earlier file are
retained. At least one filename must be supplied.

### Func ParseConfigFilesStrict
```go
func ParseConfigFilesStrict(ctx context.Context, cfg any, filenames ...string) error
```
ParseConfigFilesStrict is like ParseConfigFiles but reports an error if any
file contains unknown fields.

### Func ParseConfigStrict
```go
func ParseConfigStrict(spec []byte, cfg any) error
```
ParseConfigStrict is like ParseConfig but reports an error if there are
unknown fields in the yaml specification. Top-level mapping fields whose
values carry a YAML anchor (&name) are permitted: they exist only to provide
reusable values for alias references and are not struct fields.

### Func ParseConfigString
```go
func ParseConfigString(spec string, cfg any) error
```
ParseConfigString is like ParseConfig but for a string.

### Func ParseConfigStringStrict
```go
func ParseConfigStringStrict(spec string, cfg any) error
```
ParseConfigStringStrict is like ParseConfigString but reports an error if
there are unknown fields in the yaml specification.

### Func ParseConfigs
```go
func ParseConfigs(cfg any, specs ...[]byte) error
```
ParseConfigs merges the YAML content of each spec into cfg. Specs are
processed in order; a field present in a later spec overrides the value set
by an earlier one, while fields only in an earlier spec are retained.

### Func ParseConfigsStrict
```go
func ParseConfigsStrict(cfg any, specs ...[]byte) error
```
ParseConfigsStrict is like ParseConfigs but reports an error if any spec
contains unknown fields.

### Func ParseDeferred
```go
func ParseDeferred[T any](d *Deferred) (T, error)
```
ParseDeferred decodes the provided Deferred YAML node into a value of type
T.



## Types
### Type ByteSize
```go
type ByteSize int64
```
ByteSize represents a quantity of bytes. It can be parsed from and
marshaled to human-readable strings using either binary (KiB, MiB, GiB,
TiB) or decimal (KB, MB, GB, TB) unit suffixes. A space between the number
and unit is optional; parsing is case-insensitive. Bare integers are treated
as bytes. Floating-point values are accepted during parsing (e.g. "1.5GiB").

### Functions

```go
func ParseByteSize(s string) (ByteSize, error)
```
ParseByteSize parses s into a ByteSize. Binary (KiB, MiB, GiB, TiB) and
decimal (KB, MB, GB, TB) suffixes are supported. A space between the number
and unit is allowed; parsing is case-insensitive. A bare number is treated
as bytes. Floating-point values are rounded to the nearest byte.



### Methods

```go
func (b ByteSize) MarshalYAML() (any, error)
```


```go
func (b ByteSize) String() string
```
String returns a human-readable representation of b. It selects the largest
binary unit (TiB, GiB, MiB, KiB) that divides b evenly, then the largest
decimal unit (TB, GB, MB, KB), and falls back to "NB" when no unit divides
evenly.


```go
func (b *ByteSize) UnmarshalYAML(value *yaml.Node) error
```




### Type Deferred
```go
type Deferred yaml.Node
```
Deferred represents a YAML node that has been captured for deferred
decoding.

### Methods

```go
func (d *Deferred) Decode(v any) error
```
Decode decodes the captured YAML node into the provided value.


```go
func (d Deferred) MarshalYAML() (any, error)
```
MarshalYAML marshals Deferred as the underlying YAML node.


```go
func (d *Deferred) UnmarshalYAML(value *yaml.Node) error
```
UnmarshalYAML captures the raw YAML node for deferred decoding.


```go
func (d *Deferred) ValueFor(key string) (*yaml.Node, bool)
```
ValueFor retrieves the value associated with the specified key from a
mapping node.




### Type FlexTime
```go
type FlexTime time.Time
```
FlexTime is a time.Time that can be unmarshaled from time.RFC3339,
time.DateTime, time.TimeOnly or time.DateOnly formats. It is always
marshaled to time.RFC3339.

### Methods

```go
func (t *FlexTime) MarshalYAML() (any, error)
```


```go
func (t FlexTime) String() string
```


```go
func (t *FlexTime) UnmarshalYAML(value *yaml.Node) error
```




### Type RFC3339Time
```go
type RFC3339Time time.Time
```
RFC3339Time is a time.Time that marshals to and from RFC3339 format.

### Methods

```go
func (t *RFC3339Time) MarshalYAML() (any, error)
```


```go
func (t RFC3339Time) String() string
```


```go
func (t *RFC3339Time) UnmarshalYAML(value *yaml.Node) error
```






## Examples
### [ExampleDeferred](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc#example-Deferred)

### [ExampleDeferred_valueFor](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc#example-Deferred_valueFor)




