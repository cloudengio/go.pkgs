# Package [cloudeng.io/webapp/webassets](https://pkg.go.dev/cloudeng.io/webapp/webassets?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/webapp/webassets)](https://goreportcard.com/report/cloudeng.io/webapp/webassets)

```go
import cloudeng.io/webapp/webassets
```


## Functions
### Func NewAssets
```go
func NewAssets(prefix string, fsys fs.FS, opts ...AssetsOption) fs.FS
```
NewAssets returns an fs.FS that is configured to be optional reloaded from
the local filesystem or to be served directly from the supplied fs.FS. The
EnableReloading option is used to enable reloading. Prefix is prepended to
all names passed to the supplied fs.FS, which is typically obtained via
go:embed. See RelativeFS for more details.

### Func RelativeFS
```go
func RelativeFS(prefix string, fs fs.FS) fs.FS
```
RelativeFS wraps the supplied FS so that prefix is prepended to all of the
paths fetched from it. This is generally useful when working with webservers
where the FS containing files is created from 'assets/...' but the URL path
to access them is at the root. So /index.html can be mapped to
assets/index.html.

### Func ServeFile
```go
func ServeFile(wr io.Writer, fsys fs.FS, name string) (int, error)
```



## Types
### Type AssetsOption
```go
type AssetsOption func(a *assets)
```
AssetsOption represents an option to NewAssets.

### Functions

```go
func EnableLogging() AssetsOption
```
EnableLogging enables logging using a built in logging function.


```go
func EnableReloading(location string, after time.Time, loadNew bool) AssetsOption
```
EnableReloading enables reloading of assets from the specified location if
they have changed since 'after'; loadNew controls whether new files, ie.
those that exist only in location, are loaded as opposed. See
cloudeng.io/io/reloadfs.


```go
func UseLogger(logger func(action reloadfs.Action, name, path string, err error)) AssetsOption
```
UseLogger enables logging using the supplied logging function.







