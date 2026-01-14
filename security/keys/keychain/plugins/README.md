# Package [cloudeng.io/security/keys/keychain/plugins](https://pkg.go.dev/cloudeng.io/security/keys/keychain/plugins?tab=doc)

```go
import cloudeng.io/security/keys/keychain/plugins
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



## Functions
### Func NextID
```go
func NextID() int32
```



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



### Methods

```go
func (e Error) Error() string
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
func NewFS(pluginPath string, sysSpecific any, args ...string) *FS
```
NewFS creates a new FS instance with the specified plugin path,
system-specific data, and plugin arguments.



### Methods

```go
func (f FS) ReadFile(name string) ([]byte, error)
```


```go
func (f FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```


```go
func (f FS) WriteFile(name string, data []byte, perm fs.FileMode) error
```


```go
func (f FS) WriteFileCtx(ctx context.Context, name string, data []byte, _ fs.FileMode) error
```




### Type Request
```go
type Request struct {
	ID          int32           `json:"id,omitempty"`
	Keyname     string          `json:"keyname"`
	Write       bool            `json:"write,omitempty"`
	Contents    []byte          `json:"contents,omitempty"`
	SysSpecific json.RawMessage `json:"sys_specific,omitempty"`
}
```
Request represents the request to the keychain plugin.

### Functions

```go
func NewRequest(keyname string, sysSpecific any) (Request, error)
```
NewRequest creates a Request to read a key with the given keyname and
system-specific data. The ID is automatically generated and is unique for
each call to this function.


```go
func NewWriteRequest(keyname string, contents []byte, sysSpecific any) (Request, error)
```
NewWriteRequest creates a Request to write a key with the given keyname,
contents, and system-specific data. The ID is automatically generated and is
unique for each call to this function.



### Methods

```go
func (req Request) NewResponse(contents []byte, responseError *Error) *Response
```
NewResponse creates a Response with the given contents and error.




### Type Response
```go
type Response struct {
	ID          int32           `json:"id,omitempty"`
	Contents    []byte          `json:"contents,omitempty"`
	Error       *Error          `json:"error,omitempty"`
	SysSpecific json.RawMessage `json:"sys_specific,omitempty"`
}
```
Response represents the response from the keychain plugin.

### Functions

```go
func RunExtPlugin(ctx context.Context, binary string, req Request, args ...string) (Response, error)
```
RunExtPlugin runs an external keychain plugin with the provided request and
returns the response. binary is either a command on the PATH or an absolute
path to the plugin executable.



### Methods

```go
func (resp Response) UnmarshalSysSpecific(v any) error
```


```go
func (resp *Response) WithSysSpecific(sysSpecific any) error
```
WithSysSpecific sets the SysSpecific field of the Response to the JSON
encoding of the given sysSpecific data.







