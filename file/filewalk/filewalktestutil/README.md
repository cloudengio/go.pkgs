# Package [cloudeng.io/file/filewalk/filewalktestutil](https://pkg.go.dev/cloudeng.io/file/filewalk/filewalktestutil?tab=doc)

```go
import cloudeng.io/file/filewalk/filewalktestutil
```

Package filewalktestutil provides utilities for testing code that uses
filewalk.FS.

## Functions
### Func Scan
```go
func Scan(ctx context.Context, fs filewalk.FS, prefix string) ([]filewalk.Entry, error)
```

### Func ScanNames
```go
func ScanNames(ctx context.Context, fs filewalk.FS, prefix string) ([]string, error)
```

### Func WalkContents
```go
func WalkContents(ctx context.Context, fs filewalk.FS, roots ...string) (prefixes, names []string, err error)
```



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
NewMockFS creates a new MockFS rooted at root. All paths must start with
root.



### Methods

```go
func (m *MockFS) Base(pathname string) string
```


```go
func (m *MockFS) IsNotExist(err error) bool
```


```go
func (m *MockFS) IsPermissionError(err error) bool
```


```go
func (m *MockFS) Join(components ...string) string
```


```go
func (m *MockFS) LevelScanner(pathname string) filewalk.LevelScanner
```


```go
func (m *MockFS) Lstat(ctx context.Context, path string) (file.Info, error)
```


```go
func (m *MockFS) Open(pathname string) (fs.File, error)
```


```go
func (m *MockFS) OpenCtx(_ context.Context, pathname string) (fs.File, error)
```


```go
func (m *MockFS) Readlink(_ context.Context, _ string) (string, error)
```


```go
func (m *MockFS) Scheme() string
```


```go
func (m *MockFS) Stat(_ context.Context, pathname string) (file.Info, error)
```


```go
func (m *MockFS) String() string
```


```go
func (m *MockFS) SysXAttr(_ any, merge file.XAttr) any
```


```go
func (m *MockFS) XAttr(_ context.Context, pathname string, fi file.Info) (file.XAttr, error)
```




### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithYAMLConfig(config string) Option
```
WithYAMLConfig specifies the YAML config to use for creating a mock
filesystem.




### Type Walker
```go
type Walker struct {
	sync.Mutex
	FS       filewalk.FS
	Prefixes []string
	Names    []string
}
```

### Methods

```go
func (w *Walker) Contents(ctx context.Context, _ *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error)
```


```go
func (w *Walker) Done(_ context.Context, _ *struct{}, _ string, err error) error
```


```go
func (w *Walker) Prefix(_ context.Context, _ *struct{}, prefix string, _ file.Info, _ error) (bool, file.InfoList, error)
```







