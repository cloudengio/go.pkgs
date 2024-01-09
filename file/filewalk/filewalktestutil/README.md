# Package [cloudeng.io/file/filewalk/filewalktestutil](https://pkg.go.dev/cloudeng.io/file/filewalk/filewalktestutil?tab=doc)

```go
import cloudeng.io/file/filewalk/filewalktestutil
```

Package filewalktestutil provides utilities for testing code that uses
filewalk.FS.

## Types
### Type MockFS
```go
type MockFS struct {
	// contains filtered or unexported fields
}
```
MockFS implements filewalk.FS for testing purposes. Note that:
 1. It does not support soft links.
 2. It does not support Open on directories, instead, LevelScanner should be
    used.
 3. It only supports paths that begin with the root directory passed to
    NewMockFS.

### Functions

```go
func NewMockFS(root string, opts ...Option) (*MockFS, error)
```



### Methods

```go
func (mfs *MockFS) Base(pathname string) string
```


```go
func (mfs *MockFS) IsNotExist(err error) bool
```


```go
func (mfs *MockFS) IsPermissionError(err error) bool
```


```go
func (mfs *MockFS) Join(components ...string) string
```


```go
func (mfs *MockFS) LevelScanner(pathname string) filewalk.LevelScanner
```


```go
func (mfs *MockFS) Lstat(ctx context.Context, path string) (file.Info, error)
```


```go
func (mfs *MockFS) Open(pathname string) (fs.File, error)
```


```go
func (mfs *MockFS) OpenCtx(_ context.Context, pathname string) (fs.File, error)
```


```go
func (mfs *MockFS) Readlink(ctx context.Context, pathname string) (string, error)
```


```go
func (mfs *MockFS) Scheme() string
```


```go
func (mfs *MockFS) Stat(ctx context.Context, pathname string) (file.Info, error)
```


```go
func (mfs *MockFS) String() string
```


```go
func (mfs *MockFS) SysXAttr(existing any, merge file.XAttr) any
```


```go
func (mfs *MockFS) XAttr(ctx context.Context, pathname string, fi file.Info) (file.XAttr, error)
```




### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithYAMLConfig(config string) Option
```







