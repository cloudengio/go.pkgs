# Package [cloudeng.io/algo/lcs](https://pkg.go.dev/cloudeng.io/algo/lcs?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/algo/lcs)](https://goreportcard.com/report/cloudeng.io/algo/lcs)

```go
import cloudeng.io/algo/lcs
```

Package lcs provides implementations of algorithms to find the longest
common subsequence/shortest edit script (LCS/SES) between two slices
suitable for use with unicode/utf8 and other alphabets.

## Types
### Type DP
```go
type DP[T comparable] struct {
	// contains filtered or unexported fields
}
```
DP represents a dynamic programming based implementation
for finding the longest common subsequence and shortest
edit script (LCS/SES) for transforming A to B. See
https://en.wikipedia.org/wiki/Longest_common_subsequence_problem.
This implementation can return all LCS and SES rather than just the first
one found. If a single LCS or SES is sufficient then the Myer's algorithm
implementation is lilkey a better choice.

### Functions

```go
func NewDP[T comparable](a, b []T) *DP[T]
```
NewDP creates a new instance of DP. The implementation supports slices of
bytes/uint8, rune/int32 and int64s.



### Methods

```go
func (dp *DP[T]) AllLCS() [][]T
```
AllLCS returns all of the the longest common subsquences.


```go
func (dp *DP[T]) Fprint(out io.Writer)
```


```go
func (dp *DP[T]) LCS() []T
```
LCS returns the longest common subsquence.


```go
func (dp *DP[T]) SES() *EditScript[T]
```
SES returns the shortest edit script.




### Type Edit
```go
type Edit[T comparable] struct {
	Op   EditOp
	A, B int
	Val  T
}
```
Edit represents a single edit. For deletions, an edit specifies the index in
the original (A) slice to be deleted. For insertions, an edit specifies the
new value and the index in the original (A) slice that the new value is to
be inserted at, but immediately after the existing value if that value was
not deleted. Insertions also provide the index of the new value in the new
(B) slice. A third operation is provided, that identifies identical values,
ie. the members of the LCS and their position in the original and new
slices. This allows for the LCS to retrieved from the SES.

An EditScript that can be trivially 'replayed' to create the new slice from
the original one.

    var b []uint8
     for _, action := range actions {
       switch action.Op {
       case Insert:
         b = append(b, action.Val.(int64))
       case Identical:
         b = append(b, a[action.A])
       }
     }


### Type EditOp
```go
type EditOp int
```
EditOp represents an edit operation.

### Constants
### Insert, Delete, Identical
```go
Insert EditOp = iota
Delete
Identical

```
Values for EditOP.




### Type EditScript
```go
type EditScript[T comparable] []Edit[T]
```
EditScript represents a series of Edits.

### Methods

```go
func (es *EditScript[T]) Apply(a []T) []T
```
Apply transforms the original slice to the new slice by applying the SES.


```go
func (es *EditScript[T]) FormatHorizontal(out io.Writer, a []T)
```
FormatVertical prints a representation of the edit script across three
lines, with the top line showing the result of applying the edit, the middle
line the operations applied and the bottom line any items deleted, eg:

     CB AB AC
    -+|-||-|+
    A  C  B


```go
func (es *EditScript[T]) FormatVertical(out io.Writer, a []T)
```
FormatVertical prints a representation of the edit script with one item per
line, eg:
  - 6864772235558415538 -8997218578518345818
  - -6615550055289275125
  - -7192184552745107772 5717881983045765875


```go
func (es *EditScript[T]) Reverse() *EditScript[T]
```
Reverse returns a new edit script that is the inverse of the one supplied.
That is, of the original script would transform A to B, then the results of
this function will transform B to A.


```go
func (es *EditScript[T]) String() string
```
String implements stringer.




### Type Myers
```go
type Myers[T comparable] struct {
	// contains filtered or unexported fields
}
```
Myers represents an implementation of Myer's longest common subsequence
and shortest edit script algorithm as as documented in: An O(ND) Difference
Algorithm and Its Variations, 1986.

### Functions

```go
func NewMyers[T comparable](a, b []T) *Myers[T]
```
NewMyers returns a new instance of Myers. The implementation supports slices
of bytes/uint8, rune/int32 and int64s.



### Methods

```go
func (m *Myers[T]) LCS() []T
```
LCS returns the longest common subsquence.


```go
func (m *Myers[T]) SES() *EditScript[T]
```
SES returns the shortest edit script.






## Examples
### [ExampleDP](https://pkg.go.dev/cloudeng.io/algo/lcs?tab=doc#example-DP)

### [ExampleMyers](https://pkg.go.dev/cloudeng.io/algo/lcs?tab=doc#example-Myers)




### TODO
- cnicolaou: improve DP implementation to use only one row+column to
store lcs lengths rather than row * column.
- cnicolaou: improve the Myers implementation as described in
An O(NP) Sequence Comparison Algorithm, Wu, Manber, Myers.




