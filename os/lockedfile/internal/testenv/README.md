# Package [cloudeng.io/os/lockedfile/internal/testenv](https://pkg.go.dev/cloudeng.io/os/lockedfile/internal/testenv?tab=doc)

```go
import cloudeng.io/os/lockedfile/internal/testenv
```


## Functions
### Func HasExec
```go
func HasExec() bool
```
HasExec reports whether the current system can start new processes using
os.StartProcess or (more commonly) exec.Command.

### Func MustHaveExec
```go
func MustHaveExec(t testing.TB)
```
MustHaveExec checks that the current system can start new processes using
os.StartProcess or (more commonly) exec.Command. If not, MustHaveExec calls
t.Skip with an explanation.




