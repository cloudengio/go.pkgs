# Package [cloudeng.io/sync/patterns](https://pkg.go.dev/cloudeng.io/sync/patterns?tab=doc)

```go
import cloudeng.io/sync/patterns
```

Package patterns provides common synchronization and communication patterns
built using channels and other primitives.

## Constants
### DefaultFIFOSize
```go
DefaultFIFOSize = 100

```

### DefaultPubSubCapacity
```go
DefaultPubSubCapacity = 100

```



## Types
### Type FIFO
```go
type FIFO[T any] struct {
	// contains filtered or unexported fields
}
```
FIFO is a goroutine-safe queue that drops the oldest item when the internal
buffer (capacity items) is full. b.out is unbuffered; items are only
delivered when a receiver is ready.

The internal state (buf, head, tail, count) is a ring buffer accessed
exclusively by the run goroutine, so drop-oldest is atomic with respect to
external readers and requires no allocations after the initial make.

### Functions

```go
func NewFIFO[T any](ctx context.Context, capacity int) *FIFO[T]
```
NewFIFO creates a new FIFO with the specified buffer capacity. If capacity
is <= 0, it defaults to DefaultFIFOSize.



### Methods

```go
func (b *FIFO[T]) In() chan<- T
```


```go
func (b *FIFO[T]) Out() <-chan T
```


```go
func (b *FIFO[T]) Stop(ctx context.Context)
```




### Type PubSub
```go
type PubSub[T any] struct {
	// contains filtered or unexported fields
}
```
PubSub provides a concurrent pub-sub mechanism that drops the oldest items
for slow subscribers when their buffer is full.

### Functions

```go
func New[T any]() *PubSub[T]
```
New returns a new PubSub instance.



### Methods

```go
func (ps *PubSub[T]) Close()
```
Close closes the PubSub instance and all of its active subscribers.


```go
func (ps *PubSub[T]) Publish(item T)
```
Publish sends an item to all active subscribers. If a subscriber's buffer is
full, its oldest item is dropped to make room for the new one. Subscribers
whose run goroutine has exited (e.g. context cancelled) are detected via
their alive channel and pruned from the map without blocking.


```go
func (ps *PubSub[T]) Subscribe(ctx context.Context, capacity int) *Subscriber[T]
```
Subscribe creates and returns a new Subscriber with the given buffer
capacity. If capacity is <=0, it defaults to DefaultPubSubCapacity. ctx is
passed to the underlying FIFO.


```go
func (ps *PubSub[T]) Unsubscribe(sub *Subscriber[T])
```
Unsubscribe removes a subscriber and closes its underlying channel.




### Type Subscriber
```go
type Subscriber[T any] struct {
	// contains filtered or unexported fields
}
```
Subscriber represents a subscription to a PubSub instance.

### Methods

```go
func (s *Subscriber[T]) C() <-chan T
```
C returns the underlying receive-only channel for use in select statements.







