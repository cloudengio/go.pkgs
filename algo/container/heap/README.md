# Package [cloudeng.io/algo/container/heap](https://pkg.go.dev/cloudeng.io/algo/container/heap?tab=doc)

```go
import cloudeng.io/algo/container/heap
```

Package heap contains various implementations of heap containers.

## Types
### Type KeyedInt64
```go
type KeyedInt64 struct {
	// contains filtered or unexported fields
}
```
KeyedInt64 implements a heap whose values include both a key and value to
allow for updates to existing items in the heap. It also keeps a running sum
of the all of the values currently in the heap, supports both ascending and
desencding operations and is safe for concurrent use.

### Functions

```go
func NewKeyedInt64(order Order) *KeyedInt64
```
NewKeyedInt64 returns a new instance of KeyedInt64.



### Methods

```go
func (ki *KeyedInt64) GobDecode(buf []byte) error
```
GobDecode implements gob.GobDecoder.


```go
func (ki *KeyedInt64) GobEncode() ([]byte, error)
```
GobEncode implements gob.GobEncode.


```go
func (ki *KeyedInt64) Len() int
```
Len returns the number of items in the heap.


```go
func (ki *KeyedInt64) MarshalJSON() ([]byte, error)
```
MarshalJSON implements json.Marshaler.


```go
func (ki *KeyedInt64) Pop() (string, int64)
```
Pop removes the top most value (either largest or smallest) from the heap.


```go
func (ki *KeyedInt64) Remove(key string)
```
Remove removes the specified item from the heap.


```go
func (ki *KeyedInt64) Sum() int64
```
Sum returns the current sum of all values in the heap.


```go
func (ki *KeyedInt64) TopN(n int) []struct {
	K string
	V int64
}
```
TopN removes at most the top most n items from the heap.


```go
func (ki *KeyedInt64) UnmarshalJSON(buf []byte) error
```
UnmarshalJSON implements json.Unmarshaler.


```go
func (ki *KeyedInt64) Update(key string, value int64)
```
Update updates the value associated with key or it adds it to the heap.




### Type Order
```go
type Order bool
```
Order determines if the heap is maintained in ascending or descending order.

### Constants
### Ascending, Descending
```go
Ascending Order = false
Descending Order = true

```
Values for Order.







