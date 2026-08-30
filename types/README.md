# Package [cloudeng.io/types](https://pkg.go.dev/cloudeng.io/types?tab=doc)

```go
import cloudeng.io/types
```

Package types provides support for working with go types.

## Functions
### Func TypeName
```go
func TypeName[T any]() string
```
TypeName returns the fully qualified type name of type T (e.g.
"example.com/pkg.MyType"). Pointer indirection is removed so that
TypeName[*MyType]() == TypeName[MyType](). A composite type that has no name
of its own, such as []MyType, is described in terms of the fully qualified
names of the types it is built from (e.g. "[]example.com/pkg.MyType").
Results are computed once per distinct concrete type and cached.

### Func TypeNameForValue
```go
func TypeNameForValue(v any) string
```
TypeNameForValue returns the fully qualified type name of v (e.g.
"example.com/pkg.MyType"). Pointer indirection is removed so that
TypeNameForValue(&MyType{}) == TypeNameForValue(MyType{}). A composite type
that has no name of its own is described in terms of the fully qualified
names of the types it is built from, as per TypeName. The empty string
is returned only for an untyped nil, which carries no type. Results are
computed once per distinct concrete type and cached.



## Types
### Type Registry
```go
type Registry sync.Map
```
Registry records types by name so that new instances of them can be
constructed given the name of the type. Any number of types may be
registered with a single Registry. It is safe for concurrent use and its
zero value is ready for use; it must not be copied after first use.

### Methods

```go
func (r *Registry) New(typeName string) (any, bool)
```
New returns a newly allocated instance of the type registered under
typeName, as a pointer to that type, or false if no type is registered under
that name. Each call returns a distinct instance.


```go
func (r *Registry) RegisterType[T any, PT *T]()
```
RegisterType registers T under the name reported by TypeName[T], replacing
any type previously registered under that name. New will construct a PT,
that is a *T, for that name.







