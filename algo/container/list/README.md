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





