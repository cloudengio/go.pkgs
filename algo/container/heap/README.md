# Package [cloudeng.io/algo/container/heap](https://pkg.go.dev/cloudeng.io/algo/container/heap?tab=doc)

```go
import cloudeng.io/algo/container/heap
```


## Functions
### Func Fix
```go
func Fix[T Value[T]](h []T, i int)
```
Fix is like heap.Fix.



## Types
### Type Heap
```go
type Heap[T Value[T]] []T
```
Heap is a generic heap implementation that can be used with any type that
implements the Value interface. It provides methods to push, pop, remove,
and fix elements in the heap. The heap is implemented as a slice in the same
manner as the standard library's heap package, but it is generic and can
work with any type that satisfies the Value interface.

### Methods

```go
func (h Heap[T]) Init()
```


```go
func (h Heap[T]) Len() int
```


```go
func (h *Heap[T]) Pop() T
```
Pop is like heap.Pop.


```go
func (h *Heap[T]) Push(x T)
```
Push is like heap.Push.


```go
func (h *Heap[T]) Remove(i int) T
```
Remove is like heap.Remove.




### Type MinMax
```go
type MinMax[K Ordered, V any] struct {
	Keys []K
	Vals []V
	// contains filtered or unexported fields
}
```
MinMax represents a min-max heap as described in:
https://liacs.leidenuniv.nl/~stefanovtp/courses/StudentenSeminarium/Papers/AL/SMMH.pdf.
Note that this requires the use of a dummy root node in the key and value
slices, ie. Keys[0] and Values[0] is always empty.

### Functions

```go
func NewMinMax[K Ordered, V any](opts ...Option[K, V]) *MinMax[K, V]
```
NewMinMax creates a new instance of MinMax.



### Methods

```go
func (h *MinMax[K, V]) Len() int
```
Len returns the number of items stored in the heap, excluding the dummy root
node.


```go
func (h *MinMax[K, V]) PopMax() (K, V)
```
PopMax removes and returns the largest key/value pair from the heap.


```go
func (h *MinMax[K, V]) PopMin() (K, V)
```
PopMin removes and returns the smallest key/value pair from the heap.


```go
func (h *MinMax[K, V]) Push(k K, v V)
```
Push pushes the key/value pair onto the heap.


```go
func (h *MinMax[K, V]) PushMaxN(k K, v V, n int)
```
PushMaxN pushes the key/value pair onto the heap if the key is greater than
than the current maximum whilst ensuring that the heap is no larger than n.


```go
func (h *MinMax[K, V]) PushMinN(k K, v V, n int)
```
PushMinN pushes the key/value pair onto the heap if the key is less than the
current minimum whilst ensuring that the heap is no larger than n.


```go
func (h *MinMax[K, V]) Remove(i int) (k K, v V)
```
Remove removes the i'th item from the heap, note that i includes the
dummy root, i.e. i == 0 is the dummy root, 1 is the min, 2 is the max etc.
Deleting the dummy root has no effect.


```go
func (h *MinMax[K, V]) Update(i int, k K, v V)
```
Update updates the i'th item in the heap, note that i includes the dummy
root element. This is more efficient than removing and adding an item.




### Type Option
```go
type Option[K Ordered, V any] func(*options[K, V])
```
Option represents the options that can be passed to NewMin and NewMax.

### Functions

```go
func WithCallback[K Ordered, V any](fn func(iv, jv V, i, j int)) Option[K, V]
```
WithCallback provides a callback function that is called after every
operation with the values and indices of the elements that have changed
location. Note that is not sufficient to track removal of items and hence
any applications that requires such tracking should do so explicitly by
wrapping the Pop operations and deleting the retried item from their data
structures.


```go
func WithData[K Ordered, V any](keys []K, vals []V) Option[K, V]
```
WithData sets the initial data for the heap.


```go
func WithSliceCap[K Ordered, V any](n int) Option[K, V]
```
WithSliceCap sets the initial capacity of the slices used to hold keys and
values.




### Type Ordered
```go
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | string
}
```
Orderded represents the set of types that can be used as keys in a heap.


### Type T
```go
type T[K Ordered, V any] struct {
	Keys []K
	Vals []V
	// contains filtered or unexported fields
}
```
T represents a heap of keys and values.

### Functions

```go
func NewMax[K Ordered, V any](opts ...Option[K, V]) *T[K, V]
```
NewMax creates a new heap with descending order.


```go
func NewMin[K Ordered, V any](opts ...Option[K, V]) *T[K, V]
```
NewMin creates a new heap with ascending order.



### Methods

```go
func (h *T[K, V]) Len() int
```
Len returns the number of elements in the heap.


```go
func (h *T[K, V]) Pop() (K, V)
```
Pop removes and returns the top element from the heap.


```go
func (h *T[K, V]) Push(k K, v V)
```
Push adds a new key and value to the heap.


```go
func (h *T[K, V]) Remove(i int) (K, V)
```
Remove removes the i'th element from the heap.


```go
func (h *T[K, V]) Update(pos int, k K, v V)
```
Update updates the key and value for the i'th element in the heap. It is
more efficient than Remove followed by Push.




### Type Value
```go
type Value[T any] interface {
	Less(x T) bool
}
```
Value represents the interface that must be implemented by types that can be
used as values in the generic Heap implementation. It requires a Less method
that compares the current instance with another instance of the same type
and returns true if the current instance is less than the other instance.




## Examples
### [ExampleHeap](https://pkg.go.dev/cloudeng.io/algo/container/heap?tab=doc#example-Heap)




