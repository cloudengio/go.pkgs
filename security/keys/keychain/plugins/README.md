# Package [cloudeng.io/security/keys/keychain/plugins](https://pkg.go.dev/cloudeng.io/security/keys/keychain/plugins?tab=doc)

```go
import cloudeng.io/security/keys/keychain/plugins
```

Package plugins defines the request/response protocol used to communicate
with an out-of-process keychain plugin, and the client side of that protocol
(see FS and RunExtPlugin).

Requests are versioned so that a client built against a newer version of
this package can detect a plugin binary that is too old to understand
its requests; this matters because the client and the plugin are
separate binaries that are frequently built and installed independently
of each other. Requests created by NewRequest and NewWriteRequest carry
RequestCurrentVersion. A plugin must call Request.CheckVersion on every
request it decodes and, if a non-nil error is returned, send that error
back as the response's Error rather than attempting to service the request.
Responses are not versioned: a plugin only ever replies to a request whose
version it has accepted.

## Constants
### RequestVersion1, RequestCurrentVersion
```go
// RequestVersion1 is the initial version of the request format.
RequestVersion1 int32 = 1
// RequestCurrentVersion is the version of the request format used by
// requests created by this package. A plugin built against this package
// must accept requests at this version or lower and reject requests with
// a higher version, see Request.CheckVersion.
RequestCurrentVersion = RequestVersion1

```



## Variables
### ErrKeyExists
```go
ErrKeyExists = NewErrorKeyExists("")

```
ErrKeyExists can be used as the target of errors.Is to check for a key
already exists error.

### ErrKeyNotFound
```go
ErrKeyNotFound = NewErrorKeyNotFound("")

```
ErrKeyNotFound can be used as the target of errors.Is to check for a key not
found error.

### ErrReadOnly
```go
ErrReadOnly = errors.New("read-only FS")

```
ErrReadOnly is returned when attempting to write to a read-only FS.

### ErrUnsupportedVersion
```go
ErrUnsupportedVersion = NewErrorUnsupportedVersion(0)

```
ErrUnsupportedVersion can be used as the target of errors.Is to check for an
unsupported request version error.



## Functions
### Func NextID
```go
func NextID() int32
```

### Func WriteRequest
```go
func WriteRequest(msgr *jsonmsgs.Messager, req Request) error
```
WriteRequest writes a Request as a framed jsonmsgs message containing a
jsonpayload typed message.

### Func WriteResponse
```go
func WriteResponse(msgr *jsonmsgs.Messager, resp Response) error
```
WriteResponse writes a Response as a framed jsonmsgs message containing a
jsonpayload typed message.



## Types
### Type Error
```go
type Error struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Stderr  string `json:"-"` // Stderr is the stder output from the plugin and is

}
```
Error represents an error returned by a plugin.

### Functions

```go
func AsError(err error) *Error
```
AsError attempts to convert the given error to a *Error and returns it.
If the error is not a *Error, it returns nil.


```go
func NewErrorKeyExists(keyname string) *Error
```
NewErrorKeyExists creates a new Error indicating that the specified key
already exists that is compatible with errors.Is and ErrorKeyExists.


```go
func NewErrorKeyNotFound(keyname string) *Error
```
NewErrorKeyNotFound creates a new Error indicating that the specified key
was not found that is compatible with errors.Is and ErrorKeyNotFound.


```go
func NewErrorUnsupportedVersion(version int32) *Error
```
NewErrorUnsupportedVersion creates a new Error indicating that the request's
version is newer than this implementation supports, that is compatible with
errors.Is and ErrUnsupportedVersion.



### Methods

```go
func (e *Error) Error() string
```


```go
func (e *Error) Is(target error) bool
```




### Type FS
```go
type FS struct {
	// contains filtered or unexported fields
}
```
FS implements a plugin-based file system for keychain that implements
file.ReadFileFS and file.WriteFileFS.

### Functions

```go
func NewFS(pluginPath string, writable bool, pluginSpecific any, args ...string) *FS
```
NewFS creates a new FS instance with the specified plugin path, writable
flag, plugin-specific data, and plugin arguments. The plugin-specific data
is passed to the plugin in the request.



### Methods

```go
func (f *FS) PluginPath() string
```


```go
func (f FS) ReadFile(name string) ([]byte, error)
```


```go
func (f FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```


```go
func (f *FS) WithLogger(logger *slog.Logger) *FS
```
WithLogger returns a new FS instance with the provided logger.


```go
func (f FS) WriteFile(name string, data []byte, perm fs.FileMode) error
```


```go
func (f FS) WriteFileCtx(ctx context.Context, name string, data []byte, _ fs.FileMode) error
```




### Type Request
```go
type Request struct {
	Version        int32          `json:"version,omitempty"`
	ID             int32          `json:"id,omitempty"`
	Keyname        string         `json:"keyname"`
	Write          bool           `json:"write,omitempty"`
	Contents       []byte         `json:"contents,omitempty"`
	PluginSpecific jsontext.Value `json:"plugin_specific,omitempty"`
}
```
Request represents the request to the keychain plugin.

### Functions

```go
func NewRequest(keyname string, pluginSpecific any) (Request, error)
```
NewRequest creates a Request to read a key with the given keyname and
system-specific data. The ID is automatically generated and is unique for
each call to this function.


```go
func NewWriteRequest(keyname string, contents []byte, pluginSpecific any) (Request, error)
```
NewWriteRequest creates a Request to write a key with the given keyname,
contents, and plugin-specific data. The ID is automatically generated and is
unique for each call to this function.


```go
func ReadRequest(msgr *jsonmsgs.Messager) (Request, error)
```
ReadRequest reads a Request from msgr as a framed jsonmsgs message
containing a jsonpayload typed message.



### Methods

```go
func (req Request) CheckVersion() *Error
```
CheckVersion verifies that the request's version can be handled by this
implementation of the plugin protocol. Requests with a version at or below
RequestCurrentVersion are accepted (including a zero version from clients
that predate versioning) and nil is returned. Requests with a newer version
return an error compatible with ErrUnsupportedVersion that plugins should
send back to the client as the response's Error.


```go
func (req *Request) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (req Request) NewResponse(contents []byte, responseError *Error) *Response
```
NewResponse creates a Response with the given contents and error.


```go
func (req *Request) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```


```go
func (req Request) UnmarshalPluginSpecific(v any) error
```
UnmarshalPluginSpecific unmarshals the plugin-specific data of the Request
into the provided value v.




### Type Response
```go
type Response struct {
	ID             int32          `json:"id,omitempty"`
	Contents       []byte         `json:"contents,omitempty"`
	Stderr         string         `json:"-"` // Stderr is the stder output from the plugin and is filled in by RunExtPlugin.
	Error          *Error         `json:"error,omitempty"`
	PluginSpecific jsontext.Value `json:"plugin_specific,omitempty"`
}
```
Response represents the response from the keychain plugin.

### Functions

```go
func ReadResponse(msgr *jsonmsgs.Messager) (Response, error)
```
ReadResponse reads a Response from msgr as a framed jsonmsgs message
containing a jsonpayload typed message.


```go
func RunExtPlugin(ctx context.Context, binary string, req Request, args ...string) (Response, error)
```
RunExtPlugin runs an external keychain plugin with the provided request and
returns the response. binary is either a command on the PATH or an absolute
path to the plugin executable.



### Methods

```go
func (resp *Response) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (resp *Response) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```


```go
func (resp Response) UnmarshalPluginSpecific(v any) error
```


```go
func (resp *Response) WithPluginSpecific(pluginSpecific any) error
```
WithPluginSpecific sets the PluginSpecific field of the Response to the JSON
encoding of the given pluginSpecific data.







