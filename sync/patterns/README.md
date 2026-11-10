# Package [cloudeng.io/sync/patterns](https://pkg.go.dev/cloudeng.io/sync/patterns?tab=doc)

```go
import cloudeng.io/sync/patterns
```

Package patterns provides common synchronization and communication patterns
built using channels and other primitives.

## Constants
### DefaultFIFOSize
```go
const DefaultFIFOSize = 100
```

### DefaultPubSubCapacity
```go
const DefaultPubSubCapacity = 100
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
func NewFIFO[T any](ctx context.Context, capacity int, opts ...Option[T]) *FIFO[T]
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




### Type Option
```go
type Option[T any] func(*options[T])
```
Option configures a FIFO created with NewFIFO.

### Functions

```go
func WithPeriodicScan[T any](interval time.Duration, remove func(item T) bool) Option[T]
```
WithPeriodicScan configures the FIFO to invoke remove for every buffered
item every interval, removing (dropping) each item for which remove returns
true. Items are visited oldest-to-newest and removed items are compacted out
of the buffer in place, preserving the FIFO order of those that remain.

This can be used to implement expiration: store an enqueue time (or
deadline) alongside each item and have remove report whether it has elapsed.

remove runs on the FIFO's internal goroutine, serialized with delivery and
drop-oldest, so it must not call back into the FIFO (In, Out, or Stop)
and should not block. If interval is <= 0 or remove is nil, no scan is
scheduled.




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
func NewPubSub[T any]() *PubSub[T]
```
NewPubSub returns a new PubSub instance.



### Methods

```go
func (ps *PubSub[T]) Close()
```
Close closes the PubSub instance and all of its active subscribers, and ends
any Forward that is feeding it. Calling it more than once is harmless.

Each subscriber's channel is closed once the items it has already buffered
have been received from it, so every subscriber must drain its channel for
the subscriptions to be released, see Subscribe.


```go
func (ps *PubSub[T]) Forward(ctx context.Context, ch <-chan T)
```
Forward publishes every item it receives from ch, returning when ch is
closed, when ctx is cancelled or when ps is closed. It blocks, so that the
caller decides which goroutine runs it, which is how a producer that hands
out a receive channel is connected to a PubSub:

    go ps.Forward(ctx, producer.Events())

Everything forwarded through one channel is published in the order it was
received, and so observed in that order by every subscriber, unlike the
items of concurrent calls to Publish.

Closing ch does not close ps: a PubSub may be fed by any number of sources
and only its owner knows when the last of them is finished. Items that
arrive on ch once ps has been closed are discarded, as they are when
published directly.


```go
func (ps *PubSub[T]) Publish(item T)
```
Publish sends an item to all active subscribers. If a subscriber's buffer is
full, its oldest item is dropped to make room for the new one. Subscribers
whose run goroutine has exited (e.g. context cancelled) are detected via
their alive channel and pruned from the map without blocking.

Items published by concurrent calls are not ordered with respect to one
another, and two subscribers may observe two such items in opposite orders.
Forward, which serializes what it publishes through a single channel,
is the way to give every subscriber the same order.


```go
func (ps *PubSub[T]) Subscribe(ctx context.Context, capacity int) *Subscriber[T]
```
Subscribe creates and returns a new Subscriber with the given buffer
capacity. If capacity is <=0, it defaults to DefaultPubSubCapacity.

ctx governs the goroutine that buffers items for this subscriber: cancelling
it ends the subscription, discarding whatever it had buffered and leaving
Subscriber.C open rather than closing it, see Subscriber.Done.

A subscription holds a goroutine until it ends, which it does when the
subscriber's channel is closed and drained, when ctx is cancelled, or when
the PubSub is closed and the subscriber drains the items it had buffered.
A consumer that abandons Subscriber.C without draining it must therefore
subscribe with a context it cancels, or the subscription outlives it.


```go
func (ps *PubSub[T]) Unsubscribe(sub *Subscriber[T])
```
Unsubscribe removes a subscriber and closes its underlying channel. Calling
it more than once, or for a subscriber of a closed PubSub, is harmless.

The subscriber's channel is closed once the items it has already buffered
have been received from it, so the subscriber must drain it for the
subscription to be released, see Subscribe.




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

C is closed when the subscription is released by Unsubscribe or
PubSub.Close, but not when the context passed to Subscribe is cancelled,
which abandons it instead. A consumer that cannot guarantee that context
outlives the subscription must therefore select on Done as well, or risk
waiting on C forever; ranging over C alone is only safe for a subscription
whose context is not cancelled.


```go
func (s *Subscriber[T]) Done() <-chan struct{}
```
Done returns a channel that is closed once nothing further will be delivered
to C, whatever the reason: Unsubscribe, PubSub.Close, or cancellation of the
context passed to Subscribe.

When the subscription is released by Unsubscribe or PubSub.Close, Done is
closed after every buffered item has been received from C and C itself has
been closed, so a consumer selecting on both arms still sees all of them.
When the context is cancelled instead, any buffered items are discarded and
Done is the only indication that the subscription has ended, since C is left
open.







