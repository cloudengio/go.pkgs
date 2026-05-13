# Package [cloudeng.io/sync/patterns](https://pkg.go.dev/cloudeng.io/sync/patterns?tab=doc)

```go
import cloudeng.io/sync/patterns
```

Package patterns provides common synchronization and communication patterns
built using channels and other primitives.

## Types
### Type FIFO
```go
type FIFO[T any] struct {
	// contains filtered or unexported fields
}
```
FIFO is a goroutine-safe queue that drops the oldest item when the internal
buffer (size items) is full. b.out is unbuffered; items are only delivered
when a receiver is ready. The internal []T slice is accessed exclusively
by the run goroutine, so the drop-oldest step never races with external
readers.

### Functions

```go
func NewFIFO[T any](ctx context.Context, size int) *FIFO[T]
```



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
func New[T any](capacity int) *PubSub[T]
```
New returns a new PubSub instance with the given buffer capacity for each
subscriber. capacity must be > 0.



### Methods

```go
func (ps *PubSub[T]) Close()
```
Close closes the PubSub instance and all of its active subscribers.


```go
func (ps *PubSub[T]) Publish(item T)
```
Publish sends an item to all active subscribers. If a subscriber's buffer is
full, its oldest item is dropped to make room for the new one.


```go
func (ps *PubSub[T]) Subscribe(ctx context.Context) *Subscriber[T]
```
Subscribe creates and returns a new Subscriber. ctx is passed to the
underlying FIFO.


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







