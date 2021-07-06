# Package [cloudeng.io/os/executil](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/os/executil)](https://goreportcard.com/report/cloudeng.io/os/executil)

```go
import cloudeng.io/os/executil
```

Package executil provides utilities for working with os/exec.

## Functions
### Func NewLineFilter
```go
func NewLineFilter(forward io.Writer, re *regexp.Regexp, ch chan<- []byte) io.WriteCloser
```
NewLineFilter returns an io.WriteCloser that scans the contents of the
supplied io.Writer and sends lines that match the regexp to the supplied
channel. It can be used to filter the output of a command started by the
exec package for example for specific output. Call Close on the returned
io.WriteCloser to ensure that all resources are reclaimed.



## Examples
### [Example](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc#example-)




### TODO
- cnicolaou: make sure all goroutines shutdown.




