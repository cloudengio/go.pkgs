# Package [cloudeng.io/encoding/json/jsonerr](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc)

```go
import cloudeng.io/encoding/json/jsonerr
```

Package jsonerr provides support for sending errors over the wire as JSON.

An encoded error carries two things: a message, which any receiver can use,
and, when the error has state that can be encoded, a typed payload that
a receiver which has registered the error's type can decode back into the
original concrete error. The type of an error is identified by its fully
qualified name (e.g. "example.com/pkg.MyError") and the payload uses the
representation defined by cloudeng.io/encoding/json/jsonpayload.

An error type is an ordinary struct with an Error method; it needs no
JSON methods of its own, since its payload is encoded and decoded by
the standard struct encoding. Types are registered for decoding with
jsonpayload.RegisterType:

    type NotFound struct {
    	Name string `json:"name"`
    }

    func (e *NotFound) Error() string { return e.Name + " not found" }

    func init() { jsonpayload.RegisterType[NotFound]() }

## Functions
### Func ErrorForWire
```go
func ErrorForWire(w Wire) (error, error)
```
ErrorForWire returns the error represented by w. If w carries a payload
whose type has been registered with jsonpayload.RegisterType then the
original concrete error is returned, so that errors.Is and errors.As can
be used on it. Otherwise an error carrying only the message is returned,
which means that an unregistered type degrades to its message rather than to
a failure. A nil error is returned for the zero Wire.

### Func Marshal
```go
func Marshal(err error) ([]byte, error)
```
Marshal encodes err as its Wire representation.

### Func Unmarshal
```go
func Unmarshal(data []byte) (error, error)
```
Unmarshal decodes an error encoded by Marshal. The outer error reports a
failure to decode; the inner one is the error that was encoded.



## Types
### Type ReadWriter
```go
type ReadWriter struct {
	Err error
}
```
ReadWriter is an error that can be both encoded and decoded, for use as an
ordinary tagged field of a struct:

    type Response struct {
    	Result string          `json:"result"`
    	Err    jsonerr.ReadWriter `json:"err"`
    }

### Methods

```go
func (rw ReadWriter) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (rw *ReadWriter) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type Reader
```go
type Reader struct {
	Err error
}
```
Reader decodes an error encoded by Writer, for use where an error is a field
of a struct that is itself decoded from JSON.

### Methods

```go
func (r *Reader) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type Wire
```go
type Wire struct {
	Message string            `json:"error"`
	Detail  *jsonpayload.Wire `json:"detail,omitempty"`
}
```
Wire is the 'on-the-wire' representation of an error and documents the
format that this package produces and accepts. Message is always present so
that a receiver can report something useful for an error whose type it does
not know. Detail is present only when the error's state could be encoded,
and is the representation used by jsonpayload.

### Functions

```go
func WireForError(err error) Wire
```
WireForError returns the Wire representation of err. Encoding the error's
state is best effort: an error such as one returned by errors.New or
fmt.Errorf has no exported state to encode, and is represented by its
message alone rather than being reported as a failure.




### Type Writer
```go
type Writer struct {
	Err error
}
```
Writer encodes an error, for use where an error is a field of a struct that
is itself encoded as JSON, or is otherwise written to an encoder.

### Methods

```go
func (w Writer) MarshalJSONTo(enc *jsontext.Encoder) error
```






## Examples
### [ExampleMarshal](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc#example-Marshal)
ExampleMarshal shows an error being sent and reconstructed as its original
concrete type, so that errors.As can be used on the far side.

### [ExampleUnmarshal](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc#example-Unmarshal)
ExampleUnmarshal shows what happens to errors that cannot be reconstructed:
one whose type is not registered by the receiver, and one that has no state
to encode in the first place. Both keep their message.

### [ExampleReadWriter](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc#example-ReadWriter)
ExampleReadWriter shows an error carried as an ordinary tagged field of a
struct that is both encoded and decoded, which is the usual case for a type
shared by both ends. Use Writer or Reader instead when only one direction is
needed, as in ExampleWriter and ExampleReader.

### [ExampleReader](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc#example-Reader)
ExampleReader shows the other half: a party that only reads errors decoding
what a Writer produced, recovering the concrete error type because it has
registered it.

### [ExampleWriter](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc#example-Writer)
ExampleWriter shows an error being encoded by a party that only writes them,
such as a server reporting a failure. Writer supplies MarshalJSONTo, so an
error can be an ordinary tagged field of a struct that is itself encoded as
JSON.




