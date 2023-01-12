# Package [cloudeng.io/file](https://pkg.go.dev/cloudeng.io/file?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file)](https://goreportcard.com/report/cloudeng.io/file)

```go
import cloudeng.io/file
```


## Types
### Type FS
```go
type FS interface {
	Open(ctx context.Context, name string) (fs.File, error)
}
```
FS is like fs.FS but with a context parameter.

### Functions

```go
func FSFromFS(fs fs.FS) FS
```
FSFromFS wraps an fs.FS to implement file.FS.




### Type WriteFS
```go
type WriteFS interface {
	FS
	Create(ctx context.Context, name string, mode fs.FileMode) (io.WriteCloser, string, error)
}
```
WriteFS extends FS to add a Create method.





