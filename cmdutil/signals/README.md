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
DebounceDuration time.Duration = time.Second

```
DebounceDuration is the time period during which subsequent identical
signals are ignored.



## Functions
### Func Defaults
```go
func Defaults() []os.Signal
```
Defaults returns a set of platform specific signals that are commonly used.

### Func NotifyWithCancel
```go
func NotifyWithCancel(ctx context.Context, signals ...os.Signal) (context.Context, func() os.Signal)
```
NotifyWithCancel is like signal.Notify except that it forks (and returns)
the supplied context to obtain a cancel function that is called when a
signal is received. It will also catch the cancelation of the supplied
context and turn into an instance of ContextDoneSignal. The returned
function can be used to wait for the signals to be received, a function is
returned to allow for the convenient use of defer. Typical usage would be:

func main() {

      ctx, wait := signals.NotifyWithCancel(context.Background(), signals.Defaults()...)
      ....
      defer wait() // wait for a signal or context cancelation.
    }

If a second, different, signal is received then os.Exit(ExitCode) is called.
Subsequent signals are the same as the first are ignored for one second but
after that will similarly lead to os.Exit(ExitCode) being called.



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







