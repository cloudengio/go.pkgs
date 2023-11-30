# Package [cloudeng.io/file/filewalk/localfs](https://pkg.go.dev/cloudeng.io/file/filewalk/localfs?tab=doc)

```go
import cloudeng.io/file/filewalk/localfs
```


## Functions
### Func New
```go
func New() filewalk.FS
```

### Func NewLevelScanner
```go
func NewLevelScanner(path string) filewalk.LevelScanner
```



## Types
### Type T
```go
type T struct{}
```
T represents an instance of filewalk.FS for a local filesystem.

### Methods

```go
func (l *T) IsNotExist(err error) bool
```


```go
func (l *T) IsPermissionError(err error) bool
```


```go
func (l *T) Join(components ...string) string
```


```go
func (l *T) LevelScanner(prefix string) filewalk.LevelScanner
```


```go
func (l *T) Lstat(_ context.Context, path string) (file.Info, error)
```


```go
func (l *T) Open(path string) (fs.File, error)
```


```go
func (l *T) OpenCtx(_ context.Context, path string) (fs.File, error)
```


```go
func (l *T) Readlink(_ context.Context, path string) (string, error)
```


```go
func (l *T) Scheme() string
```


```go
func (l *T) Stat(_ context.Context, path string) (file.Info, error)
```







