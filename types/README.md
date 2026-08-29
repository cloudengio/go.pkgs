# Package [cloudeng.io/types](https://pkg.go.dev/cloudeng.io/types?tab=doc)

```go
import cloudeng.io/types
```

Package types provides support for working with gp types.

## Functions
### Func TypeName
```go
func TypeName[T any]() string
```
TypeName returns the fully qualified type name of type T (e.g.
"example.com/pkg.MyType"). Pointer indirection is removed so that
TypeName[*MyType]() == TypeName[MyType](). Results are computed once per
distinct concrete type and cached.

### Func TypeNameForValue
```go
func TypeNameForValue(v any) string
```
TypeNameForValue returns the fully qualified type name of v (e.g.
"example.com/pkg.MyType"). Pointer indirection is removed so that
TypeNameForValue(&MyType{}) == TypeNameForValue(MyType{}). Results are
computed once per distinct concrete type and cached.




