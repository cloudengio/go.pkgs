# Package [cloudeng.io/file/localfs](https://pkg.go.dev/cloudeng.io/file/localfs?tab=doc)

```go
import cloudeng.io/file/localfs
```


## Functions
### Func NewLevelScanner
```go
func NewLevelScanner(path string, openwait time.Duration) filewalk.LevelScanner
```



## Types
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
func (f *T) XAttr(_ context.Context, name string, info file.Info) (file.XAttr, error)
```







