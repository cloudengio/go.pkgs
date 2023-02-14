# Package [cloudeng.io/sync/errgroup](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc)

```go
import cloudeng.io/sync/errgroup
```

Package errgroup simplifies common patterns of goroutine use, in particular
making it straightforward to reliably wait on parallel or pipelined
goroutines, exiting either when the first error is encountered or waiting
for all goroutines to finish regardless of error outcome. Contexts are
used to control cancelation. It is modeled on golang.org/x/sync/errgroup
and other similar packages. It makes use of cloudeng.io/errors to simplify
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

T may be instantiated directly, in which case, all go routines will run
to completion and all errors will be collected and made available vie the
Errors field and the return value of Wait. Alternatively WithContext can be
used to create Group with an embedded cancel function that will be called
once either when the first error occurs or when Wait is called. WithCancel
behaves like WithContext but allows both the context and cancel function
to be supplied which is required for working with context.WithDeadline and
context.WithTimeout.

### Functions

```go
func WithCancel(cancel func()) *T
```
WithCancel returns a new T that will call the supplied cancel function once
on either a first non-nil error being returned or when Wait is called.


```go
func WithConcurrency(g *T, n int) *T
```
WithConcurrency returns a new Group that will limit the number of goroutines
to n. Note that the Go method will block when this limit is reached.
A value of 0 for n implies no limit on the number of goroutines to use.


```go
func WithContext(ctx context.Context) (*T, context.Context)
```
WithContext returns a new Group that will call the cancel function derived
from the supplied context once on either a first non-nil error being
returned by a goroutine or when Wait is called.



### Methods

```go
func (g *T) Go(f func() error)
```
Go runs the supplied function from a goroutine. If this group was created
using WithLimit then Go will block until a goroutine is available.


```go
func (g *T) GoContext(ctx context.Context, f func() error)
```
GoContext is a drop-in alternative to the Go method that checks for
ctx.Done() before calling g.Go. If the ctx has been canceled it will return
immediately recoding the error and calling the internal stored cancel
function.


```go
func (g *T) Wait() error
```
Wait waits for all goroutines to finish.






## Examples
### [ExampleT](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T)

### [ExampleT_parallel](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T_parallel)

### [ExampleT_pipeline](https://pkg.go.dev/cloudeng.io/sync/errgroup?tab=doc#example-T_pipeline)




