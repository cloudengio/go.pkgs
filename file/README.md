# Package [cloudeng.io/file](https://pkg.go.dev/cloudeng.io/file?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file)](https://goreportcard.com/report/cloudeng.io/file)

```go
import cloudeng.io/file
```


## Types
### Type FS
```go
type FS interface {
	fs.FS
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}
```
FS extends fs.FS with OpenCtx.

### Functions

```go
func WrapFS(fs fs.FS) FS
```
WrapFS wraps an fs.FS to implement file.FS.




### Type Info
```go
type Info struct {
	// contains filtered or unexported fields
}
```
Info implements fs.FileInfo with gob and json encoding/decoding. Note that
the Sys value is not encoded/decode and is only avalilable within the
process that originally created the info Instance.

### Functions

```go
func NewInfo(name string, size int64, mode fs.FileMode, mod time.Time, dir bool, sys interface{}) *Info
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
func (fi *Info) IsDir() bool
```


```go
func (fi *Info) MarshalJSON() ([]byte, error)
```


```go
func (fi *Info) ModTime() time.Time
```


```go
func (fi *Info) Mode() fs.FileMode
```


```go
func (fi *Info) Name() string
```


```go
func (fi *Info) Size() int64
```


```go
func (fi *Info) Sys() interface{}
```


```go
func (fi *Info) UnmarshalJSON(data []byte) error
```







