# Package [cloudeng.io/algo/codec](https://pkg.go.dev/cloudeng.io/algo/codec?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/algo/codec)](https://goreportcard.com/report/cloudeng.io/algo/codec)

```go
import cloudeng.io/algo/codec
```

Package codec provides support for interpreting byte slices as slices of
other basic types such as runes, int64's or strings. Go's lack of generics
make this awkward and this package currently supports a fixed set of basic
types (slices of byte/uint8, rune/int32, int64 and string).

## Types
### Type Decoder
```go
type Decoder interface {
	Decode(input []byte) interface{}
}
```
Decoder represents the ability to decode a byte slice into a slice of some
other data type.

### Functions

```go
func NewDecoder(fn interface{}, opts ...Option) (Decoder, error)
```
NewDecode returns an instance of Decoder appropriate for the supplied
function. The currently supported function signatures are:

    func([]byte) (uint8, int)
    func([]byte) (int32, int)
    func([]byte) (int64, int)
    func([]byte) (string, int)




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
of unused to used capacity exceeds the specified percentage. That is, if
cap(slice) - len(slice)) / len(slice) exceeds the percentage new underlying
storage is allocated and contents copied. The default value for
ResizePercent is 100.






## Examples
### [ExampleDecoder](https://pkg.go.dev/cloudeng.io/algo/codec?tab=doc#example-Decoder)




