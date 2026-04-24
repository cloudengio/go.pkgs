# Package [cloudeng.io/sync/ctxsync](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc)

```go
import cloudeng.io/sync/ctxsync
```

Package ctxsync provides context aware synchronisation primitives.

## Types
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
returns, the task is removed from the WaitGroup. f must not panic.


```go
func (wg *WaitGroup) Wait(ctx context.Context)
```
Wait blocks until the WaitGroup counter reaches zero or the context is
canceled, whichever comes first.






## Examples
### [ExampleWaitGroup](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc#example-WaitGroup)




