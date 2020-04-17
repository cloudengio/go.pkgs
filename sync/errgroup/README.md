# Package [cloudeng.io/sync/errgroup](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/sync/errgroup)](https://goreportcard.com/report/cloudeng.io/sync/errgroup)

```go
import cloudeng.io/sync/errgroup
```

Package errgroup simplifies common patterns of goroutine use, in particular
making it straightforward to reliably wait on parallel or pipelined
goroutines, exiting either when the first error is encountered or waiting
for all goroutines to finish regardless of error outcome. Contexts are used
to control cancelation. It is modeled on golang.org/x/sync/errgroup and
other similar packages. It makes use of cloudeng.io/errors to simplify
collecting multiple errors.

## Types
### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents a set of goroutines working on some common coordinated sets of
tasks.

T may be instantiated directly, in which case, all go routines will run to
completion and all errors will be collected and made available vie the
Errors field and the return value of Wait. Alternatively WithContext can be
used to create Group with an embedded cancel function that will be called
once either when the first error occurs or when Wait is called. WithCancel
behaves like WithContext but allows both the context and cancel function to
be supplied which is required for working with context.WithDeadline and
context.WithTimeout.



## Examples

### [ExampleT](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T)

### [ExampleT_parallel](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T_parallel)

### [ExampleT_pipeline](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T_pipeline)



