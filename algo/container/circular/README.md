# Package [cloudeng.io/algo/container/circular](https://pkg.go.dev/cloudeng.io/algo/container/circular?tab=doc)

```go
import cloudeng.io/algo/container/circular
```

Package circular provides 'circular' data structures,

## Types
### Type Buffer
```go
type Buffer[T any] struct {
	// contains filtered or unexported fields
}
```
Buffer provides a circular buffer that grows as needed.

### Functions

```go
func NewBuffer[T any](size int) *Buffer[T]
```
NewBuffer creates a new buffer with the specified initial size.



### Methods

```go
func (b *Buffer[T]) Append(v []T)
```
Append appends the specified values to the buffer, growing the buffer as
needed.


```go
func (b *Buffer[T]) Cap() int
```
Cap returns the current capacity of the buffer.


```go
func (b *Buffer[T]) Compact()
```
Compact reduces the storage used by the buffer to the minimum necessary to
store its current contents. This also has the effect of freeing any pointers
that are no longer accessible via the buffer and hence may be GC'd.


```go
func (b *Buffer[T]) Head(n int) []T
```
Head returns the first n elements of the buffer, removing them from the
buffer. If n is greater than the number of elements in the buffer then all
elements are returned. The values returned are not zeroed out and hence if
pointers will not be GC'd until the buffer itself is released or Compact is
called.


```go
func (b *Buffer[T]) Len() int
```
Len returns the current number of elements in the buffer.







