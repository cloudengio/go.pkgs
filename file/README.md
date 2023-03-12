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
Info extends fs.FileInfo to provide additional information such
as user/group, symbolic link status etc, as well gob and json
encoding/decoding. Note that the Sys value is not encoded/decoded and
is only avalilable within the process that originally created the info
Instance.

### Functions

```go
func NewInfo(name string, size int64, mode fs.FileMode, modTime time.Time,
	options InfoOption) *Info
```
NewInfo creates a new instance of Info.



### Methods

```go
func (fi *Info) GobDecode(data []byte) error
```


```go
func (fi *Info) GobEncode() ([]byte, error)
```


```go
func (fi *Info) Group() string
```
Group returns the group associated with the file.


```go
func (fi *Info) IsDir() bool
```
IsDir implements fs.FileInfo.


```go
func (fi *Info) IsLink() bool
```
IsLink returns true if the file is a symbolic link.


```go
func (fi *Info) MarshalJSON() ([]byte, error)
```


```go
func (fi *Info) ModTime() time.Time
```
ModTime implements fs.FileInfo.


```go
func (fi *Info) Mode() fs.FileMode
```
Mode implements fs.FileInfo.


```go
func (fi *Info) Name() string
```
Name implements fs.FileInfo.


```go
func (fi *Info) Size() int64
```
Size implements fs.FileInfo.


```go
func (fi *Info) Sys() interface{}
```
Sys implements fs.FileInfo.


```go
func (fi *Info) UnmarshalJSON(data []byte) error
```


```go
func (fi *Info) User() string
```
User returns the user associated with the file.




### Type InfoOption
```go
type InfoOption struct {
	User    string
	Group   string
	IsDir   bool
	IsLink  bool
	SysInfo interface{}
}
```
InfoOption is used to provide additional fields when creating an Info
instance using NewInfo.





