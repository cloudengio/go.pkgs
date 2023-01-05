# Package [cloudeng.io/file/filetestutil](https://pkg.go.dev/cloudeng.io/file/filetestutil?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/filetestutil)](https://goreportcard.com/report/cloudeng.io/file/filetestutil)

```go
import cloudeng.io/file/filetestutil
```


## Functions
### Func Contents
```go
func Contents(fs fs.FS) map[string][]byte
```
Contents returns the contents stored in the mock fs.FS.

### Func NewFile
```go
func NewFile(rd io.ReadCloser, info fs.FileInfo) fs.File
```

### Func NewInfo
```go
func NewInfo(name string, size int, mode fs.FileMode, mod time.Time, dir bool, sys interface{}) fs.FileInfo
```

### Func NewMockFS
```go
func NewMockFS(opts ...FSOption) fs.FS
```



## Types
### Type BufferCloser
```go
type BufferCloser struct {
	*bytes.Buffer
}
```

### Methods

```go
func (rc *BufferCloser) Close() error
```




### Type FSOption
```go
type FSOption func(o *fsOptions)
```

### Functions

```go
func FSErrorOnly(err error) FSOption
```


```go
func FSWithRandomContents(src rand.Source, maxSize int) FSOption
```


```go
func FSWithRandomContentsAfterRetry(src rand.Source, maxSize, numRetries int, err error) FSOption
```







