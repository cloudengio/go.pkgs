# Package [cloudeng.io/encoding/json/jsonpayload](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc)

```go
import cloudeng.io/encoding/json/jsonpayload
```


## Functions
### Func Decode
```go
func Decode[T Unmarshaler](dec *jsontext.Decoder, val T) error
```
Decode reads a typed message from dec into val, which must be non-nil. The
message's type name must be that of T, and the payload is decoded into val
in place, so that decoding allocates nothing. The type argument is inferred
from val, and no registration is required since the expected type is known
at compile time.

### Func NewInstance
```go
func NewInstance(typeName string) (any, bool)
```
NewInstance returns a newly allocated instance of the type registered under
typeName, as a pointer to that type, or false if no type is registered under
that name. It allows another package to decode the payload of a message for
itself, rather than through the readers here.

### Func RegisterType
```go
func RegisterType[T any, PT *T]()
```
RegisterType registers T under the name reported by TypeName[T], replacing
any type previously registered under that name. It is used by ReaderAny to
construct a new instance of the type when decoding a message.



## Types
### Type PointerReaderWriter
```go
type PointerReaderWriter[T any] interface {
	comparable
	*T
	ReaderWriter
}
```
PointerReaderWriter constrains PT to be a pointer to T that implements
ReaderWriter, which is how ReadWriter both encodes and decodes the value it
holds without allocating one.


### Type ReadWriter
```go
type ReadWriter[T any, PT PointerReaderWriter[T]] struct {
	Value T
}
```
ReadWriter is a typed message that can be both encoded and decoded, for use
as a field of a struct that is itself marshaled as JSON:

    type Envelope struct {
    	ID      int                                     `json:"id"`
    	Message jsonpayload.ReadWriter[Greeting, *Greeting] `json:"message"`
    }

Unlike Reader, the value is held by the field rather than pointed to by it,
so a zero valued Envelope can be unmarshaled into directly: the payload
is decoded in place and nothing is allocated. Both type arguments must be
given, since Go infers type arguments for calls but not for types.

### Functions

```go
func NewReadWriter[T any, PT PointerReaderWriter[T]](val T) ReadWriter[T, PT]
```
NewReadWriter returns a ReadWriter holding val, for use when encoding.



### Methods

```go
func (rw ReadWriter[T, PT]) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (rw *ReadWriter[T, PT]) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type ReadWriterAny
```go
type ReadWriterAny struct {
	Value ReaderWriter
}
```
ReadWriterAny is a typed message that can be both encoded and decoded when
its type is not known at compile time, for use as a field of a struct
that is itself marshaled as JSON. Every type that may be decoded must be
registered with RegisterType; Value must be non-nil to encode.

### Functions

```go
func NewReadWriterAny(val ReaderWriter) ReadWriterAny
```
NewReadWriterAny returns a ReadWriterAny holding val, for use when encoding.



### Methods

```go
func (rw ReadWriterAny) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (rw *ReadWriterAny) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type Reader
```go
type Reader[T Unmarshaler] struct {
	Value T
}
```
Reader is a JSON decoder for typed messages that adapts Decode to the
json.UnmarshalerFrom interface, for when a typed message appears within a
larger JSON document, or is decoded with json.Unmarshal. To decode a message
on its own, call Decode directly and avoid the wrapper entirely.

Value is supplied by the caller and decoded into in place, so that decoding
allocates nothing; it must be non-nil. Decoding twice into the same Reader
decodes into the same value both times.

### Functions

```go
func NewReader[T Unmarshaler](val T) Reader[T]
```
NewReader returns a Reader that decodes into val. The type argument is
inferred from val.



### Methods

```go
func (r Reader[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error
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




### Type ReaderWriter
```go
type ReaderWriter interface {
	json.MarshalerTo
	json.UnmarshalerFrom
}
```
ReaderWriter is a value that can both encode and decode itself as the
payload of a typed message.


### Type Unmarshaler
```go
type Unmarshaler interface {
	comparable
	json.UnmarshalerFrom
}
```
Unmarshaler is the constraint for a value that a typed message can be
decoded into. It requires comparable so that a missing value can be detected
by comparison with the zero value rather than by reflection; a decode target
is in practice a pointer, which is always comparable.


### Type Wire
```go
type Wire struct {
	Type    string         `json:"type"`
	Payload jsontext.Value `json:"payload"`
}
```
Wire represents the 'on-the-wire' representation of a typed message and
documents the format that this package produces and accepts. Another package
can marshal a Wire value to generate a message that the readers here will
understand, or unmarshal a message into one to inspect it, without depending
on the writers in this package.

The readers and writers here work directly with the token stream rather than
with a Wire value, so as not to buffer the payload; the tests verify that
what they produce and accept is exactly this representation.


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




### Type WriterAny
```go
type WriterAny struct {
	Value json.MarshalerTo
}
```
WriterAny is a JSON encoder for typed messages whose type is not known at
compile time, such as messages held in a slice of some interface type.
The name written is that of the value's dynamic type, whereas Writer uses
the name of its type parameter, which for an interface typed variable would
name the interface rather than the message it holds. Value must be non-nil.

### Functions

```go
func NewWriterAny(val json.MarshalerTo) WriterAny
```
NewWriterAny returns a WriterAny for val.



### Methods

```go
func (w WriterAny) MarshalJSONTo(enc *jsontext.Encoder) error
```






## Examples
### [ExampleDecode](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-Decode)
ExampleDecode shows the simplest way to read a typed message whose type is
known at compile time. Nothing needs to be registered, and the message is
decoded directly into the caller's value.

### [ExamplePointerReaderWriter](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-PointerReaderWriter)
ExamplePointerReaderWriter shows what a message type must provide to be
carried by a ReadWriter. The methods must be declared on the pointer type,
since decoding has to modify the value: Greeting, declared above,
satisfies PointerReaderWriter[Greeting] by declaring MarshalJSONTo and
UnmarshalJSONFrom on *Greeting. It also shows a ReadWriter used on its own
rather than as a field of an enclosing struct.

### [ExampleReadWriter](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-ReadWriter)
ExampleReadWriter shows a typed message carried as an ordinary tagged field
of a struct, when its type is known at compile time. Both type arguments
must be spelled out, since Go infers type arguments for calls but not for
types. Nothing needs to be registered and nothing is allocated: the payload
is decoded into the field in place.

### [ExampleReadWriterAny](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-ReadWriterAny)
ExampleReadWriterAny shows the same, for a field whose message type is not
known until it is read. Each type that may be carried must be registered.

### [ExampleReader](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-Reader)
ExampleReader shows Reader adapting Decode to json.UnmarshalerFrom, so that
a typed message can be read with json.Unmarshal. The value to decode into is
supplied by the caller and is filled in place.

### [ExampleReader_nested](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-Reader_nested)
ExampleReader_nested shows the case that Reader exists for: a typed message
carried as one field of a larger document, decoded by json/v2 itself.
A type that implements json.UnmarshalerFrom itself can call Decode instead
and do without the Reader.

### [ExampleReaderAny](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-ReaderAny)
ExampleReaderAny shows how to read messages whose type is not known until
the message is read. Every type that may be encountered must be registered
so that ReaderAny can construct one.

### [ExampleWriter](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-Writer)
ExampleWriter shows how a value is written as a typed message: an object
carrying the fully qualified name of the value's type alongside its payload.

### [ExampleWriterAny](https://pkg.go.dev/cloudeng.io/encoding/json/jsonpayload?tab=doc#example-WriterAny)
ExampleWriterAny shows how to write messages that are reached through
a variable of interface type, such as a slice of mixed message types.
WriterAny names each value's own type, whereas Writer would name the
interface, leaving a reader with no way to recover the message's type.




