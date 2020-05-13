# Package [cloudeng.io/algo/lcs/textdiff](https://pkg.go.dev/cloudeng.io/algo/lcs/textdiff?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/algo/lcs/textdiff)](https://goreportcard.com/report/cloudeng.io/algo/lcs/textdiff)

```go
import cloudeng.io/algo/lcs/textdiff
```

Package textdiff providers support for diff'ing text.

## Functions
### Func DP
```go
func DP(a, b interface{}) lcs.EditScript
```
DP uses cloudeng.io/algo/myers to generate diffs.

### Func LineFNVHashDecoder
```go
func LineFNVHashDecoder(data []byte) (string, int64, int)
```
LineFNVHashDecoder decodes a byte slice into newline delimited blocks each
of which is represented by a 64 bit hash obtained from fnv.New64a.

### Func Myers
```go
func Myers(a, b interface{}) lcs.EditScript
```
Myers uses cloudeng.io/algo/myers to generate diffs.



## Types
### Type Diff
```go
type Diff struct {
	// contains filtered or unexported fields
}
```
Diff represents the ability to diff two slices.

### Functions

```go
func DiffByLines(a, b []byte) *Diff
```
DiffByLines calls DiffByLinesUsing with the Myers function.


```go
func DiffByLinesUsing(a, b []byte, engine func(a, b interface{}) lcs.EditScript) *Diff
```
DiffByLinesUsing diffs the supplied strings on a line-by-line basis using
the supplied function to generate the diffs.



### Methods

```go
func (d *Diff) Group(i int) *Group
```
Group returns the i'th 'diff group'.


```go
func (d *Diff) NumGroups() int
```
NumGroups returns the number of 'diff groups' created.


```go
func (d *Diff) Same() bool
```
Same returns true if there were no diffs.




### Type Group
```go
type Group struct {
	// contains filtered or unexported fields
}
```
Group represents a single diff 'group', that is a set of
insertions/deletions that are pertain to the same set of lines.

### Methods

```go
func (g *Group) Deleted() string
```
Deleted returns the text would be deleted.


```go
func (g *Group) Inserted() string
```
Inserted returns the text to be inserted.


```go
func (g *Group) Summary() string
```
Summary returns a summary message in the style of the unix/linux diff
command line tool, eg. 1,2a3.




### Type LineDecoder
```go
type LineDecoder struct {
	// contains filtered or unexported fields
}
```
LineDecoder represents a decoder that can be used to split a byte stream
into lines for use with the cloudeng.io/algo/lcs package.

### Functions

```go
func NewLineDecoder(fn func(data []byte) (string, int64, int)) *LineDecoder
```
NewLineDecoder returns a new instance of LineDecoder.



### Methods

```go
func (ld *LineDecoder) Decode(data []byte) (int64, int)
```
Decode can be used as the decode function when creating a new decoder using
cloudeng.io/algo.codec.NewDecoder.


```go
func (ld *LineDecoder) Line(i int) (string, uint64)
```
Line returns the i'th line.


```go
func (ld *LineDecoder) NumLines() int
```
NumLines returns the number of lines decoded.







### TODO
- cnicolaou: adjust the lcs algorithms to be identical to diff?




