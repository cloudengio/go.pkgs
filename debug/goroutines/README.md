# Package [cloudeng.io/debug/goroutines](https://pkg.go.dev/cloudeng.io/debug/goroutines?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/debug/goroutines)](https://goreportcard.com/report/cloudeng.io/debug/goroutines)

```go
import cloudeng.io/debug/goroutines
```


## Functions
### Func Format
```go
func Format(gs ...*Goroutine) string
```
Format formats Goroutines back into the normal string representation.



## Types
### Type Frame
```go
type Frame struct {
	Call   string
	File   string
	Line   int64
	Offset int64
}
```
Frame represents a single stack frame.


### Type Goroutine
```go
type Goroutine struct {
	ID      int64
	State   string
	Stack   []*Frame
	Creator *Frame
}
```
Goroutine represents a single goroutine.

### Functions

```go
func Get(ignore ...string) ([]*Goroutine, error)
```
Get gets a set of currently running goroutines and parses them into a
structured representation. Any goroutines that match the ignore list are
ignored.


```go
func Parse(buf []byte, ignore ...string) ([]*Goroutine, error)
```
Parse parses a stack trace into a structure representation.







