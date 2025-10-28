# Package [cloudeng.io/os/executil](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc)

```go
import cloudeng.io/os/executil
```

Package executil provides utilities for working with os/exec.

## Functions
### Func ExecName
```go
func ExecName(path string) string
```
ExecName returns path in a form suitable for use as an executable. For unix
systems the path is unchanged. For windows a '.exe' suffix is added if not
already present.

### Func GoBuild
```go
func GoBuild(ctx context.Context, binary string, args ...string) (string, error)
```

### Func NewLineFilter
```go
func NewLineFilter(forward io.Writer, ch chan<- []byte, res ...*regexp.Regexp) io.WriteCloser
```
NewLineFilter returns an io.WriteCloser that scans the contents of the
supplied io.Writer and sends lines that match the regexp to the supplied
channel. It can be used to filter the output of a command started by the
exec package for example for specific output. If no regexps are supplied,
all lines are sent. Close must be called on the returned io.WriteCloser to
ensure that all resources are reclaimed.



## Examples
### [Example](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc#example-)




### TODO
- cnicolaou: make sure all goroutines shutdown.




