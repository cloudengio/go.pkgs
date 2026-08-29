# Package [cloudeng.io/encoding/json/jsonpayload](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc)

```go
import cloudeng.io/encoding/json/jsonpayload
```


## Functions
### Func RegisterType
```go
func RegisterType[T any, PT *T]()
```
RegisterType registers T under the name reported by TypeName[T], replacing
any type previously registered under that name. It is used by ReaderAny to
construct a new instance of the type when decoding a message.



## Types
### Type Reader
```go
type Reader[T json.UnmarshalerFrom] struct {
	Value T
}
```
Reader is a JSON decoder for typed messages. It should be used when the
expected type is known at compile time, in which case no registration
is required. Value is supplied by the caller and decoded into in place,
so that decoding allocates nothing; it must be non-nil. Decoding twice into
the same Reader decodes into the same value both times.

### Functions

```go
func NewReader[T json.UnmarshalerFrom](val T) Reader[T]
```
NewReader returns a Reader that decodes into val. The type argument is
inferred from val.



### Methods

```go
func (r *Reader[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type ReaderAny
```go
type ReaderAny struct {
	Value json.UnmarshalerFrom
}
```
ReaderAny is a JSON decoder for typed messages. It should be used when the
expected type is not known at compile time. The Value field will be set to
the decoded value. All types that may be decoded must be registered with
RegisterType.

### Methods

```go
func (r *ReaderAny) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type Wrapper
```go
type Wrapper[T any] struct {
	Value T
}
```
Wrapper adapts any json-marshalable type T to the
json.MarshalerTo and json.UnmarshalerFrom interfaces.
It is not intended for performance-sensitive code. Also note
that its type name will include the full generic type name, e.g.
"cloudeng.io/encoding/json/jsonpayload.Wrapper[example.com/pkg.MyType]".

### Methods

```go
func (j *Wrapper[T]) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (j *Wrapper[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type Writer
```go
type Writer[T json.MarshalerTo] struct {
	Value T
}
```
Writer is a JSON encoder for typed messages. T is the type of the value
being encoded, which is typically a pointer type since MarshalJSONTo is
usually implemented on a pointer receiver; the type name written to the
message is that of the value type either way, since TypeName removes pointer
indirection.

### Functions

```go
func NewWriter[T json.MarshalerTo](val T) Writer[T]
```
NewWriter returns a Writer for val. The type argument is inferred from val.



### Methods

```go
func (w Writer[T]) MarshalJSONTo(enc *jsontext.Encoder) error
```







