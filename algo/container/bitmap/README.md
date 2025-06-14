# Package [cloudeng.io/algo/container/bitmap](https://pkg.go.dev/cloudeng.io/algo/container/bitmap?tab=doc)

```go
import cloudeng.io/algo/container/bitmap
```


## Types
### Type T
```go
type T []uint64
```
T is a bitmap type that represents a set of bits using a slice of uint64.

### Functions

```go
func New(size int) T
```
New creates a new bitmap of the specified size in bits. The size must
be greater than zero. The bitmap is represented as a slice of uint64.
The caller must keep track of size if it cares that the size of the bitmap
is rounded up to the nearest multiple of 64 bits.



### Methods

```go
func (b T) AllClear(start, size int) iter.Seq[int]
```
AllClear returns an iterator over all clear bits in the bitmap starting from
the specified index and never exceeding the specified size or size of the
bitmap itself.


```go
func (b T) AllSet(start, size int) iter.Seq[int]
```
AllSet returns an iterator over all set bits in the bitmap starting from the
specified index and never exceeding the specified size or size of the bitmap
itself.


```go
func (b T) Clear(i int)
```
Clear clears the bit at index i in the bitmap, setting it to 0. If i is out
of bounds, the function does nothing.


```go
func (b T) ClearUnsafe(i int)
```
ClearUnsafe clears the bit at index i in the bitmap without bounds checking.


```go
func (b T) IsSet(i int) bool
```
IsSet checks if the bit at index i in the bitmap is set (1). If i is out of
bounds, it returns false.


```go
func (b T) IsSetUnsafe(i int) bool
```
IsSetUnsafe checks if the bit at index i in the bitmap is set (1) without
bounds checking.


```go
func (b T) MarshalJSON() ([]byte, error)
```


```go
func (b T) NextClear(start, size int) int
```
NextClear returns the index of the next clear bit in the bitmap starting
from the specified index and never exceeding the specified size or size of
the bitmap itself.


```go
func (b T) NextSet(start, size int) int
```
NextSet returns the index of the next set bit in the bitmap starting from
the specified index and never exceeding the specified size or size of the
bitmap itself.


```go
func (b T) Set(i int)
```
Set sets the bit at index i in the bitmap to 1. If i is out of bounds,
the function does nothing.


```go
func (b T) SetUnsafe(i int)
```
SetUnsafe sets the bit at index i in the bitmap to 1 without bounds
checking.


```go
func (b *T) UnmarshalJSON(data []byte) error
```







