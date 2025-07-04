# Package [cloudeng.io/file/localfs](https://pkg.go.dev/cloudeng.io/file/localfs?tab=doc)

```go
import cloudeng.io/file/localfs
```


## Constants
### DefaultLargeFileBlockSize
```go
DefaultLargeFileBlockSize = 1024 * 1024 * 16 // Default block size is 16 MiB.


```



## Functions
### Func NewLevelScanner
```go
func NewLevelScanner(path string, openwait time.Duration) filewalk.LevelScanner
```



## Types
### Type LargeFile
```go
type LargeFile struct {
	// contains filtered or unexported fields
}
```
LargeFile is a wrapper around a file that supports reading large files in
blocks. It implements the largefile.Reader interface.

### Functions

```go
func NewLargeFile(file *os.File, blockSize int, digest digests.Hash) (*LargeFile, error)
```
NewLargeFile creates a new LargeFile instance that wraps the provided file
and uses the specified block size for reading. The supplied digest is simply
returned by the Digest() method and is not used to validate the file's
contents directly.



### Methods

```go
func (lf *LargeFile) ContentLengthAndBlockSize() (int64, int)
```
ContentLengthAndBlockSize implements largefile.Reader.


```go
func (lf *LargeFile) Digest() digests.Hash
```
Digest implements largefile.Reader.


```go
func (lf *LargeFile) GetReader(_ context.Context, from, _ int64) (io.ReadCloser, largefile.RetryResponse, error)
```
GetReader implements largefile.Reader.


```go
func (lf *LargeFile) Name() string
```
Name implements largefile.Reader.




### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithScannerOpenWait(d time.Duration) Option
```




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents the local filesystem. It implements FS, ObjectFS and
filewalk.FS

### Functions

```go
func New(opts ...Option) *T
```
NewLocalFS returns an instance of file.FS that provides access to the local
filesystem.



### Methods

```go
func (f *T) Base(path string) string
```


```go
func (f *T) Delete(_ context.Context, path string) error
```


```go
func (f *T) DeleteAll(_ context.Context, path string) error
```


```go
func (f *T) EnsurePrefix(_ context.Context, path string, perm fs.FileMode) error
```


```go
func (f *T) Get(_ context.Context, path string) ([]byte, error)
```


```go
func (f *T) IsNotExist(err error) bool
```


```go
func (f *T) IsPermissionError(err error) bool
```


```go
func (f *T) Join(components ...string) string
```


```go
func (f *T) LevelScanner(prefix string) filewalk.LevelScanner
```


```go
func (f *T) Lstat(_ context.Context, path string) (file.Info, error)
```


```go
func (f *T) Open(name string) (fs.File, error)
```


```go
func (f *T) OpenCtx(_ context.Context, name string) (fs.File, error)
```


```go
func (f *T) Put(_ context.Context, path string, perm fs.FileMode, data []byte) error
```


```go
func (f *T) ReadFile(name string) ([]byte, error)
```


```go
func (f *T) ReadFileCtx(_ context.Context, name string) ([]byte, error)
```


```go
func (f *T) Readlink(_ context.Context, path string) (string, error)
```


```go
func (f *T) Scheme() string
```


```go
func (f *T) Stat(_ context.Context, path string) (file.Info, error)
```


```go
func (f *T) SysXAttr(existing any, merge file.XAttr) any
```


```go
func (f *T) WriteFile(name string, data []byte, perm fs.FileMode) error
```


```go
func (f *T) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error
```


```go
func (f *T) XAttr(_ context.Context, name string, info file.Info) (file.XAttr, error)
```







