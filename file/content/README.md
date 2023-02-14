# Package [cloudeng.io/file/content](https://pkg.go.dev/cloudeng.io/file/content?tab=doc)

```go
import cloudeng.io/file/content
```

Package content provides support for working with different content types.
In particular it defines a mean of specifying content types and a registry
for matching content types against handlerspackage content

## Functions
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



## Types
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
LookupConverters returns the converters registered for converting the 'from'
content type to the 'to' content type. The returned handlers are in the same
order as that registered via RegisterConverter.


```go
func (c *Registry[T]) LookupHandlers(ctype Type) ([]T, error)
```
LookupHandlers returns the list handler registered for the given content
type.


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
func TypeForPath(path string) Type
```
TypeForPath returns the Type for the given path. The Type is determined by
obtaining the extension of the path and looking up the corresponding mime
type.







