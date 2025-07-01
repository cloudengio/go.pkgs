# Package [cloudeng.io/algo/container/list](https://pkg.go.dev/cloudeng.io/algo/container/list?tab=doc)

```go
import cloudeng.io/algo/container/list
```


## Types
### Type Double
```go
type Double[T any] struct {
	// contains filtered or unexported fields
}
```
Double provides a doubly linked list.

### Functions

```go
func NewDouble[T any]() *Double[T]
```



### Methods

```go
func (dl *Double[T]) Append(val T) DoubleID[T]
```


```go
func (dl *Double[T]) Forward() iter.Seq[T]
```


```go
func (dl *Double[T]) Head() T
```


```go
func (dl *Double[T]) Len() int
```


```go
func (dl *Double[T]) Prepend(val T) DoubleID[T]
```


```go
func (dl *Double[T]) Remove(val T, cmp func(a, b T) bool)
```


```go
func (dl *Double[T]) RemoveItem(id DoubleID[T])
```


```go
func (dl *Double[T]) RemoveReverse(val T, cmp func(a, b T) bool)
```


```go
func (dl *Double[T]) Reset()
```


```go
func (dl *Double[T]) Reverse() iter.Seq[T]
```


```go
func (dl *Double[T]) Tail() T
```




### Type DoubleID
```go
type DoubleID[T any] *doubleItem[T]
```


### Type Single
```go
type Single[T any] struct {
	// contains filtered or unexported fields
}
```
Single provides a singly linked list.

### Functions

```go
func NewSingle[T any]() *Single[T]
```
NewSingle creates a new instance of Single[T] with an initial empty state.



### Methods

```go
func (sl *Single[T]) Append(val T) SingleID[T]
```
Append adds a new item to the end of the list and returns its ID.


```go
func (sl *Single[T]) Forward() iter.Seq[T]
```
Forward returns an iterator over the list.


```go
func (sl *Single[T]) Head() T
```
Head returns the first item in the list.


```go
func (sl *Single[T]) Len() int
```
Len returns the number of items in the singly linked list.


```go
func (sl *Single[T]) Prepend(val T) SingleID[T]
```
Prepend adds a new item to the beginning of the list and returns its ID.


```go
func (sl *Single[T]) Remove(val T, cmp func(a, b T) bool)
```
Remove removes the first occurrence of the specified value from the list.


```go
func (sl *Single[T]) RemoveItem(id SingleID[T])
```
RemoveItem removes the item with the specified ID from the list.


```go
func (sl *Single[T]) Reset()
```
Reset resets the singly linked list to its initial empty state.




### Type SingleID
```go
type SingleID[T any] *singleItem[T]
```





