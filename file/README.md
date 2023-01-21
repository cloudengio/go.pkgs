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
	// Scheme returns the URI scheme that this FS supports. Scheme should
	// be "file" for local file system access.
	Scheme() string
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}
```
FS extends fs.FS with Scheme and OpenCtx.

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
process that originally created the info Instance. It also users a User and
Group methods.

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
func (fi *Info) Group() string
```
Group returns the group associated with the file.


```go
func (fi *Info) IsDir() bool
```
IsDir implements fs.FileInfo.


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
func (fi *Info) SetGroup(group string)
```
SetGroup sets the group associated with the file.


```go
func (fi *Info) SetUser(user string)
```
SetUser sets the user associated with the file.


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







