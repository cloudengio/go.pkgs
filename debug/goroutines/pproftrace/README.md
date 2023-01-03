# Package [cloudeng.io/debug/goroutines/pproftrace](https://pkg.go.dev/cloudeng.io/debug/goroutines/pproftrace?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/debug/goroutines/pproftrace)](https://goreportcard.com/report/cloudeng.io/debug/goroutines/pproftrace)

```go
import cloudeng.io/debug/goroutines/pproftrace
```


## Functions
### Func Format
```go
func Format(key, value string) (string, error)
```
Format returns a nicely formatted dump of the goroutines with the pprof
key/value label.

### Func LabelExists
```go
func LabelExists(key, value string) (bool, error)
```
LabelExists returns true if a goroutine with the pprof key/value label
exists.

### Func Run
```go
func Run(ctx context.Context, key, value string, runner func(context.Context))
```
Run uses pprof's label support to attach the specified key/value label to
all goroutines spawed by the supplied runner. RunUnderPprof returns when
runner returns.




