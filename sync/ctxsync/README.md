# Package [cloudeng.io/sync/ctxsync](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc)

```go
import cloudeng.io/sync/ctxsync
```

Package ctxsync provides context aware synchronisation primitives.

## Types
### Type WaitGroup
```go
type WaitGroup struct {
	sync.WaitGroup
}
```
WaitGroup represents a context aware sync.WaitGroup

### Methods

```go
func (wg *WaitGroup) Wait(ctx context.Context)
```
Wait blocks until the WaitGroup reaches zero or the context is canceled,
whichever comes first.






## Examples
### [ExampleWaitGroup](https://pkg.go.dev/cloudeng.io/sync/ctxsync?tab=doc#example-WaitGroup)




