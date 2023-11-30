# Package [cloudeng.io/file](https://pkg.go.dev/cloudeng.io/file?tab=doc)

```go
import cloudeng.io/file
```


## Functions
### Func ContextWithFS
```go
func ContextWithFS(ctx context.Context, container fs.ReadFileFS) context.Context
```
ContextWithFS returns a new context that contains the provided instance of
fs.ReadFileFS stored with as a valye within it.

### Func FSFromContext
```go
func FSFromContext(ctx context.Context) (fs.ReadFileFS, bool)
```
FSFromContext returns the fs.ReadFileFS instance, if any, stored within the
context.

### Func FSOpen
```go
func FSOpen(ctx context.Context, name string) (fs.File, error)
```
FSOpen will open name using the context's fs.ReadFileFS instance if one is
present, otherwise it will use os.Open.

### Func FSReadFile
```go
func FSReadFile(ctx context.Context, name string) ([]byte, error)
```
FSreadAll will read name using the context's fs.ReadFileFS instance if one
is present, otherwise it will use os.ReadFile.



## Types
### Type FS
```go
type FS interface {
	fs.FS

	// Scheme returns the URI scheme that this FS supports. Scheme should
	// be "file" for local file system access.
	Scheme() string

	// OpenCtx is like fs.Open but with a context.
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}
```
FS extends fs.FS with Scheme and OpenCtx.

### Functions

```go
func WrapFS(fs fs.FS) FS
```
WrapFS wraps an fs.FS to implement file.FS.




### Type FSFactory
```go
type FSFactory interface {
	New(ctx context.Context, scheme string) (FS, error)
	NewFromMatch(ctx context.Context, m cloudpath.Match) (FS, error)
}
```
FSFactory is implemented by types that can create a file.FS for a given
URI scheme or for a cloudpath.Match. New is used for the common case
where an FS can be created for an entire filesystem instance, whereas
NewMatch is intended for the case where more granular approach is required.
The implementations of FSFactory will typically store the authentication
credentials required to create the FS when New or NewMatch is called.
For AWS S3 for example, the information required to create an aws.Config
will be stored in used when New or NewMatch are called. New will create an
FS for S3 in general, whereas NewMatch can take more specific action such as
creating an FS for a specific bucket or region with different credentials.


### Type Info
```go
type Info struct {
	// contains filtered or unexported fields
}
```
Info implements fs.FileInfo to provide binary, gob and json
encoding/decoding. The SysInfo field is not encoded/decoded and hence is
only available for use within the process that Info was instantiated in.

### Functions

```go
func NewInfo(
	name string,
	size int64,
	mode fs.FileMode,
	modTime time.Time,
	sysInfo any) Info
```
NewInfo creates a new instance of Info.


```go
func NewInfoFromFileInfo(fi fs.FileInfo) Info
```



### Methods

```go
func (fi *Info) AppendBinary(buf *bytes.Buffer) error
```


```go
func (fi *Info) DecodeBinary(data []byte) ([]byte, error)
```
DecodeBinary decodes the supplied data into the receiver and returns the
remaining data.


```go
func (fi Info) IsDir() bool
```
IsDir implements fs.FileInfo.


```go
func (fi Info) MarshalBinary() ([]byte, error)
```
Implements encoding.BinaryMarshaler.


```go
func (fi Info) MarshalJSON() ([]byte, error)
```


```go
func (fi Info) ModTime() time.Time
```
ModTime implements fs.FileInfo.


```go
func (fi Info) Mode() fs.FileMode
```
Mode implements fs.FileInfo.


```go
func (fi Info) Name() string
```
Name implements fs.FileInfo.


```go
func (fi *Info) SetSys(i any)
```


```go
func (fi Info) Size() int64
```
Size implements fs.FileInfo.


```go
func (fi Info) Sys() any
```
Sys implements fs.FileInfo.


```go
func (fi *Info) UnmarshalBinary(data []byte) error
```
Implements encoding.BinaryUnmarshaler.


```go
func (fi *Info) UnmarshalJSON(data []byte) error
```




### Type InfoList
```go
type InfoList []Info
```
InfoList represents a list of Info instances. It provides efficient
encoding/decoding operations.

### Functions

```go
func DecodeBinaryInfoList(data []byte) (InfoList, []byte, error)
```
DecodeBinaryInfoList decodes the supplied data into an InfoList and returns
the remaining data.



### Methods

```go
func (il InfoList) AppendBinary(buf *bytes.Buffer) error
```
AppendBinary appends a binary encoded instance of Info to the supplied byte
slice.


```go
func (il InfoList) AppendInfo(info Info) InfoList
```
Append appends an Info instance to the list and returns the updated list.


```go
func (il InfoList) MarshalBinary() ([]byte, error)
```
MarshalBinary implements encoding.BinaryMarshaler.


```go
func (il *InfoList) UnmarshalBinary(data []byte) (err error)
```
UnmarshalBinary implements encoding.BinaryUnmarshaler.







