# Package [cloudeng.io/file](https://pkg.go.dev/cloudeng.io/file?tab=doc)

```go
import cloudeng.io/file
```


## Variables
### ErrNotImplemented
```go
ErrNotImplemented = fmt.Errorf("not implemented")

```
ErrNotImplemented is returned by methods that are not implemented by a
particular filesystem.

### ErrSchemeNotSupported
```go
ErrSchemeNotSupported = fmt.Errorf("scheme not supported")

```



## Functions
### Func ContextWithFS
```go
func ContextWithFS(ctx context.Context, container ...fs.ReadFileFS) context.Context
```
ContextWithFS returns a new context that contains the provided instances of
fs.ReadFileFS stored with as a value within it.

### Func FSFromContext
```go
func FSFromContext(ctx context.Context) ([]fs.ReadFileFS, bool)
```
FSFromContext returns the list of fs.ReadFileFS instances, if any, stored
within the context.

### Func FSOpen
```go
func FSOpen(ctx context.Context, filename string) (fs.File, error)
```
FSOpen will attempt to open filename using the context's set of
fs.ReadFileFS instances (if any), in the order in which they were provided
to ContextWithFS, returning the first successful result. If no fs.ReadFileFS
instances are present in the context or none successfully open the file,
then os.Open is used.

### Func FSReadFile
```go
func FSReadFile(ctx context.Context, name string) ([]byte, error)
```
FSreadFile is like FSOpen but calls ReadFile instead of Open.



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

	// Readlink returns the contents of a symbolic link.
	Readlink(ctx context.Context, path string) (string, error)

	// Stat will follow symlinks/redirects/aliases.
	Stat(ctx context.Context, path string) (Info, error)

	// Lstat will not follow symlinks/redirects/aliases.
	Lstat(ctx context.Context, path string) (Info, error)

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// Base is like filepath.Base for the filesystem supported by this filesystem.
	Base(path string) string

	// IsPermissionError returns true if the specified error, as returned
	// by the filesystem's implementation, is a result of a permissions error.
	IsPermissionError(err error) bool

	// IsNotExist returns true if the specified error, as returned by the
	// filesystem's implementation, is a result of the object not existing.
	IsNotExist(err error) bool

	// XAttr returns extended attributes for the specified file.Info
	// and file.
	XAttr(ctx context.Context, path string, fi Info) (XAttr, error)

	// SysXAttr returns a representation of the extended attributes using the
	// native data type of the underlying file system. If existing is
	// non-nil and is of that file-system specific type the contents of
	// XAttr are merged into it.
	SysXAttr(existing any, merge XAttr) any
}
```
FS extends fs.FS with Scheme and OpenCtx.


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
NewInfoFromFileInfo creates a new instance of Info from a fs.FileInfo.



### Methods

```go
func (fi *Info) AppendBinary(buf *bytes.Buffer) error
```
AppendBinary appends a binary encoded instance of Info to the supplied
buffer.


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
SetSys sets the SysInfo field. Note that the Sys field is never
encoded/decoded.


```go
func (fi Info) Size() int64
```
Size implements fs.FileInfo.


```go
func (fi Info) Sys() any
```
Sys implements fs.FileInfo. Note that the Sys field is never
encoded/decoded.


```go
func (fi Info) Type() fs.FileMode
```
Type implements fs.Entry


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




### Type ObjectFS
```go
type ObjectFS interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Put(ctx context.Context, path string, perm fs.FileMode, data []byte) error
	EnsurePrefix(ctx context.Context, path string, perm fs.FileMode) error
	Delete(ctx context.Context, path string) error
	// DeleteAll delets all objects with the specified prefix.
	DeleteAll(ctx context.Context, prefix string) error
}
```
ObjectFS represents a writeable object store. It is intended to backed
by cloud or local filesystems. The permissions may be ignored by some
implementations.


### Type ReadFileFS
```go
type ReadFileFS interface {
	ReadFile(name string) ([]byte, error)
	ReadFileCtx(ctx context.Context, name string) ([]byte, error)
}
```
ReadFileFS defines an FS style interface for reading files.


### Type WriteFileFS
```go
type WriteFileFS interface {
	WriteFile(name string, data []byte, perm fs.FileMode) error
	WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
}
```
WriteFileFS defines an FS style interface for writing files.


### Type XAttr
```go
type XAttr struct {
	UID, GID       int64  // -1 for non-posix filesystems that don't support numeric UID, GID
	User, Group    string // Used for systems that don't support numeric UID, GID
	Device, FileID uint64
	Blocks         int64
	Hardlinks      uint64
}
```
XAttr represents extended information about a directory or file as obtained
from the filesystem.

### Methods

```go
func (x XAttr) CompareGroup(o XAttr) bool
```
CompareGroup compares the GID fields if >=0 and the Group fields otherwise.


```go
func (x XAttr) CompareUser(o XAttr) bool
```
CompareUser compares the UID fields if >=0 and the User fields otherwise.







