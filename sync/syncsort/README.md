# Package [cloudeng.io/sync/syncsort](https://pkg.go.dev/cloudeng.io/sync/syncsort?tab=doc)

```go
import cloudeng.io/sync/syncsort
```

Package syncsort provides support for synchronised sorting.

## Variables
### ErrStopped
```go
ErrStopped = errors.New("sequencer stopped")

```
ErrStopped is returned by NextItem when the sequencer has been stopped via
Stop or its ordering goroutine has exited.



## Types
### Type Item
```go
type Item[T any] struct {
	V T
	// contains filtered or unexported fields
}
```
Item represents a single item in a stream that is to be ordered. It is
returned by NextItem and simply wraps the supplied type with a monotonically
increasing sequence number that determines its position in the ordered
stream. This sequence number is assigned by NextItem.


### Type Option
```go
type Option func(*options)
```
Option represents an option to NewSequencer.

### Functions

```go
func WithMaxBuffered(n int) Option
```
WithMaxBuffered limits the maximum number of unconsumed items (those
obtained via NextItem and not yet yielded by Scan) to n. When n <= 0,
buffering is unbounded.

When a positive limit is set, NextItem blocks if n unconsumed items are
already in flight until an item is consumed via Scan, the context is
cancelled, or Stop is called. Callers using WithMaxBuffered must therefore
consume items via Scan concurrently with producers calling NextItem. Because
the internal ordering goroutine continues to drain inputCh regardless of
arrival order, out-of-order arrival among in-flight items will not deadlock.




### Type Sequencer
```go
type Sequencer[T any] struct {
	// contains filtered or unexported fields
}
```
Sequencer implements a streaming sequencer that will accept a stream of
unordered items (sent to it over a channel) and allow for that stream to be
scanned in order. The end of the unordered stream is signaled by closing
this channel. Items to be sent in the stream are obtained via calls to
NextItem and the order of calls to NextItem determines the order of items
returned by the scanner.

### Functions

```go
func NewSequencer[T any](ctx context.Context, inputCh <-chan Item[T], opts ...Option) *Sequencer[T]
```
NewSequencer returns a new instance of Sequencer.



### Methods

```go
func (s *Sequencer[T]) Done() <-chan struct{}
```
Done returns a channel that is closed when the internal ordering goroutine
exits (due to Stop, context cancellation, or end of stream). Producers can
select on Done when sending items to the input channel so they do not block
if the consumer stops early.


```go
func (s *Sequencer[T]) Err() error
```
Err returns any errors encountered by the scanner.


```go
func (s *Sequencer[T]) Item() T
```
Item returns the current item available in the scanner.


```go
func (s *Sequencer[T]) NextItem(ctx context.Context, item T) (Item[T], error)
```
NextItem returns a new Item to be used with Sequencer. The order of calls
made to NextItem determines the order that they are returned by the scanner.
When WithMaxBuffered is configured, NextItem blocks until a buffer slot is
available, ctx or the sequencer's context is cancelled, or Stop is called.


```go
func (s *Sequencer[T]) Scan() bool
```
Scan returns true if the next ordered item in the stream is available to be
read via Item. On cancellation, Stop, or end of stream, Scan returns false
without yielding a zero item; call Err to inspect any error.


```go
func (s *Sequencer[T]) Stop()
```
Stop terminates the sequencer early so its internal goroutine exits without
requiring the consumer to drain remaining items from Scan and without
cancelling the context passed to NewSequencer. Any calls waiting in NextItem
return ErrStopped, and Done is closed. Stop is safe to call multiple times.







