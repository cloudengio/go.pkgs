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
func ParseConfig(spec []byte, cfg interface{}) error
```
ParseConfig will parse the yaml config in spec into the requested type.
It provides improved error reporting via ErrorWithSource.

### Func ParseConfigFile
```go
func ParseConfigFile(ctx context.Context, filename string, cfg interface{}) error
```
ParseConfigFile reads a yaml config file as per ParseConfig using
file.FSReadFile to read the file. The use of FSReadFile allows for the
configuration file to be read from storage system, including from embed.FS,
instead of the local filesystem if an instance of fs.ReadFileFS is stored in
the context.

### Func ParseConfigString
```go
func ParseConfigString(spec string, cfg interface{}) error
```
ParseConfigString is like ParseConfig but for a string.

### Func ParseConfigURI
```go
func ParseConfigURI(ctx context.Context, filename string, cfg interface{}, handlers map[string]URLHandler) error
```
ParseConfigURI is like ParseConfigFile but for a URI.

### Func WithFSForURI
```go
func WithFSForURI(ctx context.Context, uri string, handlers map[string]URLHandler) (context.Context, string)
```
WithFSForURI will parse the supplied URI and if it has a scheme that matches
one of the handlers, will call the handler to create a new context and
pathname. If no handler is found, the original context and URI are returned.



## Types
### Type FlexTime
```go
type FlexTime time.Time
```
FlexTime is a time.Time that can be unmarshaled from time.RFC3339,
time.DateTime, time.TimeOnly or time.DateOnly formats. It is always
marshaled to time.RFC3339.

### Methods

```go
func (t *FlexTime) MarshalYAML() (interface{}, error)
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
func (t *RFC3339Time) MarshalYAML() (interface{}, error)
```


```go
func (t RFC3339Time) String() string
```


```go
func (t *RFC3339Time) UnmarshalYAML(value *yaml.Node) error
```




### Type URLHandler
```go
type URLHandler func(context.Context, *url.URL) (ctx context.Context, pathname string)
```
URLHandler is a function that uses the supplied URL to create a new context
containing an fs.ReadFileFS instance that can be used to read the contents
of the original URL using the returned pathname.





