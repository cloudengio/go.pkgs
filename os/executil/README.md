# Package [cloudeng.io/os/executil](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/os/executil)](https://goreportcard.com/report/cloudeng.io/os/executil)

```go
import cloudeng.io/os/executil
```


## Functions
### Func NewLineFilter
```go
func NewLineFilter(forward io.Writer, re *regexp.Regexp, ch chan<- []byte) io.WriteCloser
```




### TODO
- cnicolaou: make sure all goroutines shutdown...




