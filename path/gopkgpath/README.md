# Package [cloudeng.io/path/gopkgpath](https://pkg.go.dev/cloudeng.io/path/gopkgpath?tab=doc)

```go
import cloudeng.io/path/gopkgpath
```

Package gopkgpath provides support for obtaining and working with go
package paths when go modules are used. It does not support vendor or GOPATH
configurations.

## Functions
### Func Caller
```go
func Caller() (string, error)
```
Caller is the same as CallerDepth(0).

### Func CallerDepth
```go
func CallerDepth(depth int) (string, error)
```
CallerDepth returns the package path of the caller at the specified depth
where a depth of 0 is the immediate caller. It determines the module name by
finding and parsing the enclosing go.mod file and as such requires that go
modules are being used.

### Func Type
```go
func Type(v any) string
```
Type returns the package path for the type of the supplied argument.
That type must be a defined/named type, anoymous types, function variables
etc will return "".




