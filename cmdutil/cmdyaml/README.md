# Package [cloudeng.io/cmdutil/cmdyaml](https://pkg.go.dev/cloudeng.io/cmdutil/cmdyaml?tab=doc)

```go
import cloudeng.io/cmdutil/cmdyaml
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

### Func ParseConfigFileStrict
```go
func ParseConfigFileStrict(ctx context.Context, filename string, cfg any) error
```
ParseConfigFileStrict is like ParseConfigFile but reports an error if there
are unknown fields in the yaml specification.

### Func ParseConfigStrict
```go
func ParseConfigStrict(spec []byte, cfg any) error
```
ParseConfigStrict is like ParseConfig but reports an error if there are
unknown fields in the yaml specification.

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

### Func ParseDeferred
```go
func ParseDeferred[T any](d *Deferred) (T, error)
```
ParseDeferred decodes the provided Deferred YAML node into a value of type
T.



## Types
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
func (d *Deferred) MarshalYAML() (any, error)
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




