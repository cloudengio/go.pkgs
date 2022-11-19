# Package [cloudeng.io/algo/codec](https://pkg.go.dev/cloudeng.io/algo/codec?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/algo/codec)](https://goreportcard.com/report/cloudeng.io/algo/codec)

```go
import cloudeng.io/algo/codec
```

Package codec provides support for interpreting byte slices as slices of
other basic types such as runes, int64's or strings.

## Types
### Type Decoder
```go
type Decoder[T any] interface {
	Decode(input []byte) []T
}
```
Decoder represents the ability to decode a byte slice into a slice of some
other data type.

### Functions

```go
func NewDecoder[T any](fn func([]byte) (T, int), opts ...Option) Decoder[T]
```
NewDecode returns an instance of Decoder appropriate for the supplied
function.




### Type Option
```go
type Option func(*options)
```
Option represents an option accepted by NewDecoder.

### Functions

```go
func ResizePercent(percent int) Option
```
ResizePercent requests that the returned slice be reallocated if the ratio
of unused to used capacity exceeds the specified percentage. That is,
if cap(slice) - len(slice)) / len(slice) exceeds the percentage new
underlying storage is allocated and contents copied. The default value for
ResizePercent is 100.


```go
func SizePercent(percent int) Option
```
SizePercent requests that the initially allocated slice be 'percent' as
large as the original input slice's size in bytes. A percent of 25 will
divide the original size by 4 for example.






## Examples
### [ExampleDecoder](https://pkg.go.dev/cloudeng.io/algo/codec?tab=doc#example-Decoder)




