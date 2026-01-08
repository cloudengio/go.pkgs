# Package [cloudeng.io/cmdutil/registry](https://pkg.go.dev/cloudeng.io/cmdutil/registry?tab=doc)

```go
import cloudeng.io/cmdutil/registry
```

Package registry provides support for various forms of registry useful for
building command line tools.

## Variables
### ErrUnknownKey
```go
ErrUnknownKey = errors.New("unregistered key")

```
ErrUnknownKey is returned when an unregistered key is encountered.



## Functions
### Func ConvertAnyArgs
```go
func ConvertAnyArgs[T any](args ...any) []T
```
ConvertAnyArgs converts a variadic list of any to a slice of the specified
type T, ignoring any arguments that are not of type T.

### Func Scheme
```go
func Scheme(path string) string
```
Scheme extracts the scheme from the given path, returning "file" if no
scheme is present.



## Types
### Type New
```go
type New[T any] func(ctx context.Context, args ...any) (T, error)
```
New is a function that creates a new instance of type T


### Type T
```go
type T[T any] struct {
	// contains filtered or unexported fields
}
```
T represents a registry for a specific type T that selected using a string
key, which is typically a URI scheme.

### Methods

```go
func (r *T[RT]) Clone() *T[RT]
```
Clone creates a shallow clone of the registry.


```go
func (r *T[T]) Get(key string) New[T]
```
Get retrieves the factory function for the given key, or nil if the key is
not registered.


```go
func (r *T[T]) Keys() []string
```
Keys returns a sorted list of all registered keys.


```go
func (r *T[T]) Register(key string, factory New[T])
```
Register registers a new factory function for the given key.







