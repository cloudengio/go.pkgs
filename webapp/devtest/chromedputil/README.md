# Package [cloudeng.io/webapp/devtest/chromedputil](https://pkg.go.dev/cloudeng.io/webapp/devtest/chromedputil?tab=doc)

```go
import cloudeng.io/webapp/devtest/chromedputil
```

Package chromedputil provides utility functions for working with the Chrome
DevTools Protocol via github.com/chromedp.

## Functions
### Func ConsoleArgsAsJSON
```go
func ConsoleArgsAsJSON(ctx context.Context, event *runtime.EventConsoleAPICalled) ([][]byte, error)
```
ConsoleArgsAsJSON converts the console API call arguments to a slice of
marshalled JSON data, one per each argument to the original console.log
call.

### Func GetRemoteObjectRef
```go
func GetRemoteObjectRef(ctx context.Context, name string) (*runtime.RemoteObject, error)
```
GetRemoteObjectRef retrieves a remote object's metadata, ie. type, object id
etc (but not it's value).

### Func GetRemoteObjectValueJSON
```go
func GetRemoteObjectValueJSON(ctx context.Context, object *runtime.RemoteObject) (*runtime.RemoteObject, jsontext.Value, error)
```
GetRemoteObjectValueJSON retrieves a remote object's (using ObjectID) value
using JSON serialization. The object is looked up by its ID and hence the
supplied ObjectID must be a reference to the object with the ObjectID field
set. Objects which already contain a JSON value will return that value
immediately. NOTE that GetRemoteObjectValueJSON will return an empty or
incomplete serialization for platform objects, the ClassName will generally
be indicative of whether the object is a platform object, e.g Response or
Promise.

### Func IsPlatformObject
```go
func IsPlatformObject(obj *runtime.RemoteObject) bool
```
IsPlatformObject returns true if the given remote object is a platform
object. The obj argument must have been obtained via a call to
GetRemoteObjectValueJSON.

### Func IsPlatformObjectError
```go
func IsPlatformObjectError(err error) bool
```
IsPlatformObjectError checks if the error is due to a platform object
serialization error. The only reliable way to determine if an object is a
platform object in chrome is to attempt a deep serialization and check for
this error.

### Func ListGlobalFunctions
```go
func ListGlobalFunctions(ctx context.Context) ([]string, error)
```
ListGlobalFunctions returns a list of all global function names defined in
the current context.

### Func Listen
```go
func Listen(ctx context.Context, handlers ...func(ctx context.Context, ev any) bool)
```
Listen sets up a listener for Chrome DevTools Protocol events and calls
each of the supplied handlers in turn when an event is received. The first
handler to return true stops the event propagation.

### Func NewAnyHandler
```go
func NewAnyHandler(ch chan<- any) func(ctx context.Context, ev any) bool
```
NewAnyHandler returns a handler for all/any events that forwards the event
to the provided channel. It should generally be the last handler in the list
passed to Listen.

### Func NewEventConsoleHandler
```go
func NewEventConsoleHandler(ch chan<- *runtime.EventConsoleAPICalled) func(ctx context.Context, ev any) bool
```
NewEventConsoleHandler returns a handler for console events that forwards
the event to the provided channel.

### Func NewEventEntryHandler
```go
func NewEventEntryHandler(ch chan<- *log.EventEntryAdded) func(ctx context.Context, ev any) bool
```
NewEventEntryHandler returns a handler for log entry events that forwards
the event to the provided channel.

### Func NewLogExceptionHandler
```go
func NewLogExceptionHandler(ch chan<- *runtime.EventExceptionThrown) func(ctx context.Context, ev any) bool
```
NewLogExceptionHandler returns a handler for log exceptions that forwards
the event to the provided channel.

### Func RunLoggingListener
```go
func RunLoggingListener(ctx context.Context, logger *slog.Logger, opts ...LoggingOption) chan struct{}
```
RunLoggingListener starts the logging listener for Chrome DevTools Protocol
events.

### Func SourceScript
```go
func SourceScript(ctx context.Context, script string) error
```
SourceScript loads a JavaScript script into the current page.

### Func WaitForPromise
```go
func WaitForPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams
```
WaitForPromise waits for a promise to resolve in the given evaluate
parameters.

### Func WithContextForCI
```go
func WithContextForCI(ctx context.Context, opts ...chromedp.ContextOption) (context.Context, func())
```
WithContextForCI returns a chromedp context that may be different on a
CI system than when running locally. The CI configuration may disable
sandboxing etc. The ExecAllocator used is created with default options (eg.
headless). Use WithExecAllocatorForCI to customize accordingly. Note that
the CI customization is in WithExecAllocatorForCI.

### Func WithExecAllocatorForCI
```go
func WithExecAllocatorForCI(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, func())
```
WithExecAllocatorForCI returns a chromedp context with an ExecAllocator that
may be configured differently on a CI system than when running locally.
The CI configuration may disable sandboxing for example.



## Types
### Type LoggingOption
```go
type LoggingOption func(*loggingOptions)
```

### Functions

```go
func WithAnyEventLogging() LoggingOption
```


```go
func WithConsoleLogging() LoggingOption
```


```go
func WithEventEntryLogging() LoggingOption
```


```go
func WithExceptionLogging() LoggingOption
```







