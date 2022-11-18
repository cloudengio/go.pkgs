# Package [cloudeng.io/cmdutil/signals](https://pkg.go.dev/cloudeng.io/cmdutil/signals?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/signals)](https://goreportcard.com/report/cloudeng.io/cmdutil/signals)

```go
import cloudeng.io/cmdutil/signals
```

Package signals provides support for working with operating system signals
and contexts.

## Constants
### ExitCode
```go
// ExitCode is the exit code passed to os.Exit when a subsequent signal is
// received.
ExitCode = 1

```



## Variables
### DebounceDuration
```go
DebounceDuration = time.Second

```
DebounceDuration is the time period during which subsequent identical
signals are ignored.



## Functions
### Func Defaults
```go
func Defaults() []os.Signal
```
Defaults returns a set of platform specific signals that are commonly used.



## Types
### Type ContextDoneSignal
```go
type ContextDoneSignal string
```
ContextDoneSignal implements os.Signal and is used to translate a canceled
context into an os.Signal as forwarded by NotifyWithCancel.

### Methods

```go
func (ContextDoneSignal) Signal()
```
Signal implements os.Signal.


```go
func (s ContextDoneSignal) String() string
```
Stringimplements os.Signal.




### Type Handler
```go
type Handler struct {
	// contains filtered or unexported fields
}
```
Handler represents a signal handler that can be used to wait for signal
reception or context cancelation as per NotifyWithCancel. In addition it
can be used to register additional cancel functions to be invoked on signal
reception or context cancelation.

### Functions

```go
func NotifyWithCancel(ctx context.Context, signals ...os.Signal) (context.Context, *Handler)
```
NotifyWithCancel is like signal.Notify except that it forks (and returns)
the supplied context to obtain a cancel function that is called when a
signal is received. It will also catch the cancelation of the supplied
context and turn it into an instance of ContextDoneSignal. The returned
handler can be used to wait for the signals to be received and to register
additional cancelation functions to be invoked when a signal is received.
Typical usage would be:

    func main() {
       ctx, handler := signals.NotifyWithCancel(context.Background(), signals.Defaults()...)
       ....
       handler.RegisterCancel(func() { ... })
       ...
       defer hanlder.WaitForSignal() // wait for a signal or context cancelation.
     }

If a second, different, signal is received then os.Exit(ExitCode) is called.
Subsequent signals are the same as the first are ignored for one second but
after that will similarly lead to os.Exit(ExitCode) being called.



### Methods

```go
func (h *Handler) RegisterCancel(fns ...func())
```
RegisterCancel registers one or more cancel functions to be invoked when a
signal is received or the original context is canceled.


```go
func (h *Handler) WaitForSignal() os.Signal
```
WaitForSignal will wait for a signal to be received. Context cancelation is
translated into a ContextDoneSignal signal.







