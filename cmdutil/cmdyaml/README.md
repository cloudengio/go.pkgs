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
### Func Expand
```go
func Expand(cfg any, mapping func(string) string)
```
ExpandEnv recursively expands environment variables in the fields of the
provided struct that have a 'yaml' tag. Embedded structs are also processed.
The provided mapping is used to look up variable values.

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
ParseConfigsStrict is like ParseConfigs but reports an error if there are
unknown fields in the yaml specification. Mapping fields at any level whose
values carry a YAML anchor (&name) are permitted.

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




### Type Option
```go
type Option func(*parserOptions)
```
Option configures a Parser.

### Functions

```go
func WithExpandMapping(fn func(string) string) Option
```
WithExpandMapping expands ${VAR} and $VAR references in the spec using fn
before parsing.


```go
func WithFS(fs file.ReadFileFS) Option
```
WithFS sets the file system used by ParseFiles. Defaults to the local OS
file system rooted at the current working directory.


```go
func WithSequenceMerge(mergeKey string) Option
```
WithSequenceMerge enables list merging via a special single-key mapping
element. When a sequence contains an element of the form {key: *anchor}
where key matches mergeKey and *anchor resolves to another sequence, the
referenced sequence's items are inlined at that position. Cross-spec anchors
are supported: an anchor defined in an earlier spec can be merged into a
sequence in a later spec. The conventional value for mergeKey is "<<",
mirroring YAML's map merge key.


```go
func WithStrictFields(strict bool) Option
```
WithStrictFields causes Parse and ParseFiles to report an error for any YAML
field that does not map to a struct field. Mapping fields at any level whose
values carry a YAML anchor (&name) are permitted.


```go
func WithYAMLVariables(mapName string) Option
```
WithYAMLVariables instructs the parser to collect scalar key-value pairs
from the named top-level mapping and expand $VAR and ${VAR} references in
specs before parsing.




### Type Parser
```go
type Parser struct {
	// contains filtered or unexported fields
}
```
Parser parses and merges YAML configurations into a destination struct,
optionally expanding environment variables and YAML-defined variables.
Create one with NewParser.

### Functions

```go
func NewParser(opts ...Option) *Parser
```
NewParser returns a Parser configured with the supplied options.



### Methods

```go
func (p *Parser) Parse(cfg any, specs ...[]byte) error
```
Parse merges the YAML content of each spec into cfg. Specs are processed in
order; a field present in a later spec overrides the value set by an earlier
one, while fields only in an earlier spec are retained.


```go
func (p *Parser) ParseFiles(ctx context.Context, cfg any, filenames ...string) error
```
ParseFiles reads and merges the YAML contents of each named file into cfg.
Files are processed in order; a field present in a later file overrides
the value set by an earlier one, while fields only in an earlier file are
retained. At least one filename must be supplied.




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




### Type Regexp
```go
type Regexp struct {
	*regexp.Regexp
}
```
Regexp wraps a *regexp.Regexp so that it can be marshaled to and unmarshaled
from YAML as the regular expression's source pattern string. The zero value
has a nil *regexp.Regexp.

### Methods

```go
func (r Regexp) MarshalYAML() (any, error)
```
MarshalYAML implements yaml.Marshaler, encoding r as its source pattern
string.


```go
func (r Regexp) String() string
```
String returns the source text of the regular expression, or "" if r wraps a
nil *regexp.Regexp.


```go
func (r *Regexp) UnmarshalYAML(value *yaml.Node) error
```
UnmarshalYAML implements yaml.Unmarshaler, compiling the YAML scalar string
value as a regular expression.




### Type RegexpList
```go
type RegexpList []Regexp
```
RegexpList is a list of Regexp values that can be marshaled to and
unmarshaled from a YAML sequence of regular expression strings.

### Methods

```go
func (rl RegexpList) MarshalYAML() (any, error)
```
MarshalYAML implements yaml.Marshaler, encoding rl as a sequence of source
pattern strings.


```go
func (rl RegexpList) Regexps() []*regexp.Regexp
```
Regexps returns a slice of the *regexp.Regexp values in rl.


```go
func (rl *RegexpList) UnmarshalYAML(value *yaml.Node) error
```
UnmarshalYAML implements yaml.Unmarshaler, compiling each element of the
YAML sequence as a regular expression.




### Type Variables
```go
type Variables struct {
	// contains filtered or unexported fields
}
```
Variables accumulates scalar key-value pairs parsed from YAML mappings.
Multiple calls to Load merge into the same map; later values overwrite
earlier ones for duplicate keys.

### Functions

```go
func NewVariables() *Variables
```



### Methods

```go
func (v *Variables) Load(spec []byte, mapName string) error
```
Load parses spec, locates the top-level YAML mapping named mapName,
and merges its entries into v. All values must be scalar (string, number, or
boolean); aggregate types (mappings, sequences) are rejected with an error.
If mapName is not present in spec Load is a no-op.


```go
func (v *Variables) Mapping(key string) string
```
Mapping returns the value stored for key, or "" if key is not present.
It is safe to call on a nil or zero-value Variables.






## Examples
### [ExampleDeferred](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc#example-Deferred)

### [ExampleDeferred_valueFor](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc#example-Deferred_valueFor)




