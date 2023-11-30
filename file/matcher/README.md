# Package [cloudeng.io/file/matcher](https://pkg.go.dev/cloudeng.io/file/matcher?tab=doc)

```go
import cloudeng.io/file/matcher
```

Package matcher provides support for matching file names, types and
modification times using boolean operators. The set of operands can be
extended by defining instances of Operand but the operators are limited to
&& and ||.

## Types
### Type Item
```go
type Item struct {
	// contains filtered or unexported fields
}
```
Item represents an operator or operand in an expression. It is exposed to
allow clients packages to create their own parsers.

### Functions

```go
func AND() Item
```
And returns an AND item.


```go
func FileType(typ string) Item
```
FileType returns a 'file type' item. It is not validated until a matcher.T
is created using New. Supported file types are (as per the unix find
command):
  - f for regular files
  - d for directories
  - l for symbolic links
  - x executable regular files

It requires that the value bein matched provides Mode() fs.FileMode or
Type() fs.FileMode (which should return Mode&fs.ModeType).


```go
func Glob(pat string, caseInsensitive bool) Item
```
Glob provides a glob operand that may be case insensitive, in which case the
value it is being against will be converted to lower case before the match
is evaluated. The pattern is not validated until a matcher.T is created
using New.


```go
func LeftBracket() Item
```
LeftBracket returns a left bracket item.


```go
func NewOperand(op Operand) Item
```
NewOperand returns an item representing an operand.


```go
func NewerThanParsed(when string) Item
```
NewerThanParsed returns a 'newer than' operand. It is not validated until
a matcher.T is created using New. The time must be expressed as one of
time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly. Due to the nature
of the parsed formats fine grained time comparisons are not possible.

It requires that the value bein matched provides ModTime() time.Time.


```go
func NewerThanTime(when time.Time) Item
```
NewerThanTime returns a 'newer than' operand with the specified time.
This should be used in place of NewerThanFormat when fine grained time
comparisons are required.

It requires that the value bein matched provides ModTime() time.Time.


```go
func OR() Item
```
OR returns an OR item.


```go
func Regexp(re string) Item
```
Regexp returns a regular expression operand. It is not compiled until a
matcher.T is created using New. It requires that the value being matched
provides Name() string.


```go
func RightBracket() Item
```
RightBracket returns a right bracket item.



### Methods

```go
func (it Item) String() string
```




### Type Operand
```go
type Operand interface {
	// Prepare is used to prepare the operand for evaluation, for example, to
	// compile a regular expression.
	Prepare() (Operand, error)
	// Eval must return false for any type that it does not support.
	Eval(any) bool
	// Needs returns true if the operand needs the specified type.
	Needs(reflect.Type) bool
	String() string
}
```
Operand represents an operand. It is exposed to allow clients packages to
define custom operands.


### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents a boolean expression of regular expressions, file type and mod
time comparisons. It is evaluated against a single input value.

### Functions

```go
func New(items ...Item) (T, error)
```
New returns a new matcher.T built from the supplied items.



### Methods

```go
func (m T) Eval(v any) bool
```
Eval evaluates the matcher against the supplied value. An empty, default
matcher will always return false.


```go
func (m T) Needs(typ any) bool
```
HasOperand returns true if the matcher's expression contains an instance of
the specified operand.


```go
func (m T) String() string
```







