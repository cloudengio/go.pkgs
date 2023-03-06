# Package [cloudeng.io/file/content](https://pkg.go.dev/cloudeng.io/file/content?tab=doc)

```go
import cloudeng.io/file/content
```

Package content provides support for working with different content types.
In particular it defines a mean of specifying content types and a registry
for matching content types against handlers for processing those types.

## Functions
### Func EncodeBinary
```go
func EncodeBinary(wr io.Writer, ctype Type, data []byte) error
```
EncodeBinary encodes the specified content.Type and byte slice as binary
data.

### Func Error
```go
func Error(err error) error
```
Error is an implementation of error that is registered with the gob package
and marshals the error as the string value returned by its Error() method.
It will return nil if the specified error is nil. Common usage is:

    response.Err = content.Error(object.Err)

### Func ParseType
```go
func ParseType(ctype Type) (string, error)
```
ParseType is like ParseTypeFull but only returns the major/minor component.

### Func ParseTypeFull
```go
func ParseTypeFull(ctype Type) (typ, par, value string, err error)
```
ParseTypeFull parses a content type specification into its major/minor
components and any parameter/value pairs. It returns an error if multiple /
or ; characters are found.

### Func WriteBinary
```go
func WriteBinary(path string, ctype Type, data []byte) error
```
WriteBinary writes the results of EncodeBinary(cytpe, data) to the specified
file.



## Types
### Type Object
```go
type Object[Value, Response any] struct {
	Type     Type
	Value    Value
	Response Response
}
```
Object represents an object/file that has been downloaded/crawled or is the
result of an API invocation. The Value field represents the typed value
of the result of the download or API operation. The Response field is the
actual response for the download, API call etc. The Response may include
additional metadata.

Object supports encoding/decoding either in binary or gob format.
The gob format assumes that the decoder knows the type of the previously
encoded binary. The binary format encodes the content.Type and a byte slice
in separately. This allows for reading the encoded data without necessarily
knowing the type of the encoded object.

When gob encoding is supported care must be taken to ensure that any fields
that are interface types are appropriately registered with the gob package.
error is a common such case and the Error function can be used to replace
the existing error with a wrapper that implements the error interface and is
registered with the gob package. Canonical usage is:

    response.Err = content.Error(object.Err)

### Methods

```go
func (o *Object[V, R]) Decode(data []byte) error
```
Decode decodes the object in data using gob.


```go
func (o *Object[V, R]) DecodeBinary(rd io.Reader) error
```
DecodeBinary will decode the object using the binary encoding format.


```go
func (o *Object[V, R]) Encode() ([]byte, error)
```
Encode encodes the object using gob.


```go
func (o *Object[V, R]) EncodeBinary(wr io.Writer) error
```
EncodeBinary will encode the object using the binary encoding format.


```go
func (o *Object[V, R]) Read(rd io.Reader) error
```
Read decodes the object using gob from the specified reader.


```go
func (o *Object[V, R]) ReadObject(path string) error
```


```go
func (o *Object[V, R]) ReadObjectBinary(path string) error
```
ReadObjectBinary will decode the object using the binary encoding format
from the specified file.


```go
func (o *Object[V, R]) Write(wr io.Writer) error
```
Write encodes the object using gob to the specified writer.


```go
func (o *Object[V, R]) WriteObject(path string) error
```
WriteObject will encode the object using gob to the specified file.


```go
func (o *Object[V, R]) WriteObjectBinary(path string) error
```
WriteObjectBinary will encode the object using the binary encoding format to
the specified file.




### Type Registry
```go
type Registry[T any] struct {
	// contains filtered or unexported fields
}
```
Registry provides a means of registering and looking up handlers for
processing content types and for converting between content types.

### Functions

```go
func NewRegistry[T any]() *Registry[T]
```
NewRegistry returns a new instance of Registry.



### Methods

```go
func (c *Registry[T]) LookupConverters(from, to Type) (T, error)
```
LookupConverters returns the converter registered for converting the 'from'
content type to the 'to' content type. The returned handlers are in the same
order as that registered via RegisterConverter.


```go
func (c *Registry[T]) LookupHandlers(ctype Type) ([]T, error)
```
LookupHandlers returns the handler registered for the given content type.


```go
func (c *Registry[T]) RegisterConverters(from, to Type, converter T) error
```
RegisterConverters registers a lust of handlers for converting from one
content type to another. The caller of LookupConverter must decide which
converter to use.


```go
func (c *Registry[T]) RegisterHandlers(ctype Type, handlers ...T) error
```
RegisterHandlers registers a handler for a given content type. The caller of
LookupHandlers must decide which converter to use.




### Type Type
```go
type Type string
```
Type represents a content type specification in mime type format,
major/minor[;parameter=value]. The major/minor part is required and the
parameter is optional. The values used need not restricted to predefined
mime types; ie. the values of major/minor;parameter=value are not restricted
to those defined by the IANA.

### Functions

```go
func Clean(ctype Type) Type
```
Clean removes any spaces around the ; separator if present. That is,
"text/plain ; charset=utf-8" becomes "text/plain;charset=utf-8".


```go
func DecodeBinary(rd io.Reader) (ctype Type, data []byte, err error)
```
DecodeBinary decodes the result of a previous call to EncodeBinary.


```go
func ReadBinary(path string) (ctype Type, data []byte, err error)
```
ReadBinary reads the contents of path and interprets them using
DecodeBinary.


```go
func TypeForPath(path string) Type
```
TypeForPath returns the Type for the given path. The Type is determined by
obtaining the extension of the path and looking up the corresponding mime
type.







