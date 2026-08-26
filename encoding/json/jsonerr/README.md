# Package [cloudeng.io/encoding/json/jsonerr](https://pkg.go.dev/cloudeng.io/encoding/json/jsonerr?tab=doc)

```go
import cloudeng.io/encoding/json/jsonerr
```

Package jsonerr provides support for working with errors sent over the
wire using JSON. It provides a registry for recreating local instances
of concrete error types that have been received from a remote process.
The type of an error is represented by its package name and type name (e.g.
"example.com/pkg.MyError").

## Functions
### Func DefaultUnknownTypeHandler
```go
func DefaultUnknownTypeHandler(err Error) error
```
DefaultUnknownTypeHandler uses errors.New(err.Error) to create an error.
The Type and Detail fields are ignored.

### Func MarshalError
```go
func MarshalError(err error) ([]byte, error)
```
MarshalError marshals an error into an Error struct suitable for
transmission over the wire. TypeNameForError(err) is used to set Error.Type,
Error.Error is set to err.Error(), and Error.Detail is set to the
JSON-encoded representation of err. Error.Type must be registered using
RegisterErrorType by the receipient of the marshaled error in order to
unmarshal the error back into its corresponding concrete type.

### Func RegisterErrorType
```go
func RegisterErrorType[T any, PT interface {
	*T
	error
}]()
```

### Func TypeNameForError
```go
func TypeNameForError(err error) string
```
TypeNameForError returns the fully qualified type name of err (e.g.
"example.com/pkg.MyError"). Returns "" for nil.

### Func UnmarshalError
```go
func UnmarshalError(data []byte) (error, error)
```
UnmarshalError expects data to be a JSON-encoded Error struct. It uses the
Type field to determine the concrete type of the error, and unmarshals the
Detail field into that type. If the Type is not registered, it returns an
error using DefaultUnknownTypeHandler.



## Types
### Type Error
```go
type Error struct {
	Error  string         `json:"error"`
	Type   string         `json:"type"`
	Detail jsontext.Value `json:"detail"`
}
```
Error represents the 'on-the-wire' error representation. All errors are must
be converted to this form before being sent over the wire, and converted
back to an error on the receiving side. The Type field is used to determine
the concrete type of the error on the receiving side, and the Detail field
contains the JSON-encoded representation of the error.

### Methods

```go
func (e *Error) MarshalJSONTo(enc *jsontext.Encoder) error
```


```go
func (e *Error) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```




### Type UnknownTypeHandler
```go
type UnknownTypeHandler func(err Error) error
```
UnknownTypeHandler is a function that handles errors of unknown types.


### Type UnmarshalErrorWithHandler
```go
type UnmarshalErrorWithHandler struct {
	// contains filtered or unexported fields
}
```
UnmarshalError implements json.UnmarshalerFrom using a custom
UnknownTypeHandler for unknown error types.

### Functions

```go
func NewUnmarshalError(handler UnknownTypeHandler) *UnmarshalErrorWithHandler
```
NewUnmarshalError creates a new UnmarshalError with the given
UnknownTypeHandler. If handler is nil, DefaultUnknownTypeHandler is used.



### Methods

```go
func (ue *UnmarshalErrorWithHandler) Unmarshal(data []byte) (error, error)
```
Unmarshal decodes data as a JSON-encoded Error and returns the concrete
Go error. The first return value is the decoded application error (or the
handler's result for unknown types); the second is any decoding failure.


```go
func (ue *UnmarshalErrorWithHandler) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```







