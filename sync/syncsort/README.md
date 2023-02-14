# Package [cloudeng.io/sync/syncsort](https://pkg.go.dev/cloudeng.io/sync/syncsort?tab=doc)

```go
import cloudeng.io/sync/syncsort
```

Package syncsort provides support for synchronised sorting.

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


### Type Sequencer
```go
type Sequencer[T any] struct {
	// contains filtered or unexported fields
}
```
Sequencer implements a streaming sequencer that will accept a stream of
unordered items (sent to it over a channel) and allow for that stream to
scanned in order. The end of the unordered stream is signaled by closing
this chanel. Items to be sent in the stream are obtained via calls to
NextItem and the order of calls to NextItem determines the order of items
returned by the scanner.

### Functions

```go
func NewSequencer[T any](ctx context.Context, inputCh <-chan Item[T]) *Sequencer[T]
```
NewSequencer returns a new instance of Sequencer.



### Methods

```go
func (s *Sequencer[T]) Err() error
```
Err returns any errors encountered by the scanner.


```go
func (s *Sequencer[T]) Item() T
```
Item returns the current item available in the scanner.


```go
func (s *Sequencer[T]) NextItem(item T) Item[T]
```
NextItem returns a new Item to be used with Sequencer. The order of calls
made to NextItem determines the order that they are returned by the scanner.


```go
func (s *Sequencer[T]) Scan() bool
```
Scan returns true of the next ordered item in the stream is available to be
reead.







