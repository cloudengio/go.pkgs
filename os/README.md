# Package [cloudeng.io/os](https://pkg.go.dev/cloudeng.io/os?tab=doc)

```go
import cloudeng.io/os
```


## Functions
### Func IsStopped
```go
func IsStopped(pid int) bool
```
IsStopped returns true if the process with the specified pid has stopped
or does not exist. Wait must have been called on the process otherwise this
function may will return true on some systems since the process may still
exist as a defunct process.

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




