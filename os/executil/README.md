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

### Func Getenv
```go
func Getenv(env []string, key string) (string, bool)
```
Getenv retrieves the value of an environment variable from the provided
slice.

### Func GoBuild
```go
func GoBuild(ctx context.Context, binary string, args ...string) (string, error)
```

### Func IsStopped
```go
func IsStopped(pid int) bool
```
IsStopped returns true if the process with the specified pid has stopped
or does not exist. Wait must have been called on the process otherwise this
function will return true on some systems since the process may still exist
as a defunct process.

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

### Func ReplaceEnvVar
```go
func ReplaceEnvVar(env []string, key, value string) []string
```
ReplaceEnvVar replaces the value of an environment variable in the provided
slice. If the variable does not exist, it is added to the slice.

### Func SignalAndWait
```go
func SignalAndWait(ctx context.Context, perSignalOrWait time.Duration, cmd *exec.Cmd, sigs ...os.Signal) error
```
SignalAndWait provides a convenience function to signal a process to
terminate by sending it one or more signals and waiting for it to terminate
but with a timeout on calling Wait and on waiting for the process to stop
after each signal. The perSignalOrWait duration is used as the timeout
for both calling Wait and for waiting for the process to stop after each
signal, hence the total time spent waiting may be up to len(sigs)+1 times
perSignalOrWait. If the process stops after any signal, SignalAndWait
returns immediately.

### Func WaitForStopped
```go
func WaitForStopped(ctx context.Context, pid int, waitFor time.Duration) error
```
WaitForStopped waits for the process with the specified pid to stop within
the specified duration. It assumes that Wait has been called.



## Examples
### [Example](https://pkg.go.dev/cloudeng.io/os/executil?tab=doc#example-)




### TODO
- cnicolaou: make sure all goroutines shutdown.




