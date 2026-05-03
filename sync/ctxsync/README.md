# Package [cloudeng.io/sync/ctxsync](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc)

```go
import cloudeng.io/sync/ctxsync
```

Package ctxsync provides context aware synchronisation primitives.

## Types
### Type SingleFlight
```go
type SingleFlight struct {
	// contains filtered or unexported fields
}
```
SingleFlight mirrors golang.org/x/sync/singleflight.Group but with different
handling of context cancelation. In particular, if a shared invocation
returns with a canceled or timedout context, but the caller's context is not
canceled, the group will reissue the invocation.

### Functions

```go
func New() *SingleFlight
```
New creates a new SingleFlight instance.



### Methods

```go
func (g *SingleFlight) Do(ctx context.Context, key string, fn func() (any, error)) (v any, err error, shared bool)
```
Do is like singleflight.Group.Do but with different handling of context
cancellation. In particular, if a shared invocation returns with a canceled
or timed out context, but the caller's context is not canceled, the group
will reissue the invocation.


```go
func (g *SingleFlight) DoChan(ctx context.Context, key string, fn func() (any, error)) <-chan singleflight.Result
```
DocChan is like singleflight.Group.DoChan but with different handling of
context cancellation. In particular, if a shared invocation returns with a
canceled or timeedout context, but the caller's context is not canceled,
the group will reissue the invocation.


```go
func (g *SingleFlight) Forget(key string)
```




### Type WaitGroup
```go
type WaitGroup struct {
	// contains filtered or unexported fields
}
```
WaitGroup is a context-aware sync.WaitGroup. The zero value is ready to use.
Unlike sync.WaitGroup, it is safe to call Add immediately after a Wait that
returned due to context cancellation.

### Methods

```go
func (wg *WaitGroup) Add(delta int)
```
Add adds delta to the WaitGroup counter. If the counter transitions from
zero to positive a new completion channel is allocated. If the counter
transitions from positive to zero all blocked Wait calls are unblocked.
It panics if the counter goes negative.


```go
func (wg *WaitGroup) Done()
```
Done decrements the WaitGroup counter by one.


```go
func (wg *WaitGroup) Go(f func())
```
Go calls f in a new goroutine and adds that task to the WaitGroup. When f
returns, the task is removed from the WaitGroup. If f panics, the task is
not removed to ensure the panic remains fatal.


```go
func (wg *WaitGroup) Wait(ctx context.Context)
```
Wait blocks until the WaitGroup counter reaches zero or the context is
canceled, whichever comes first.






## Examples
### [ExampleWaitGroup](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc#example-WaitGroup)




