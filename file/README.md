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







