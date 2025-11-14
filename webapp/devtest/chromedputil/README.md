# Package [cloudeng.io/webapp/devtest/chromedputil](https://pkg.go.dev/cloudeng.io/webapp/devtest/chromedputil?tab=doc)

```go
import cloudeng.io/webapp/devtest/chromedputil
```

Package chromedputil provides utility functions for working with the Chrome
DevTools Protocol via github.com/chromedp.

## Variables
### AllocatorOptsForCI
```go
// AllocatorOptsForCI are the default ExecAllocator options for CI environments,
// they extend chromedp.DefaultExecAllocatorOptions.
AllocatorOptsForCI = []chromedp.ExecAllocatorOption{

	chromedp.NoFirstRun,
	chromedp.NoDefaultBrowserCheck,
	chromedp.Flag("headless", "new"),

	chromedp.Flag("disable-background-networking", true),

	chromedp.Flag("disable-background-timer-throttling", true),
	chromedp.Flag("disable-backgrounding-occluded-windows", true),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-client-side-phishing-detection", true),
	chromedp.Flag("disable-default-apps", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("disable-extensions", true),
	chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
	chromedp.Flag("disable-hang-monitor", true),
	chromedp.Flag("disable-ipc-flooding-protection", true),
	chromedp.Flag("disable-popup-blocking", true),
	chromedp.Flag("disable-prompt-on-repost", true),
	chromedp.Flag("disable-renderer-backgrounding", true),
	chromedp.Flag("disable-sync", true),
	chromedp.Flag("force-color-profile", "srgb"),
	chromedp.Flag("metrics-recording-only", true),
	chromedp.Flag("safebrowsing-disable-auto-update", true),
	chromedp.Flag("enable-automation", true),
	chromedp.Flag("password-store", "basic"),
	chromedp.Flag("use-mock-keychain", true),

	chromedp.DisableGPU,
	chromedp.NoSandbox,
	chromedp.Flag("disable-setuid-sandbox", true),
	chromedp.Flag("headless", "new"),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-crash-reporter", true),
	chromedp.Flag("disable-component-update", true),
	chromedp.Flag("disable-features", "MetricsReporting,UserMetrics"),
}

```



## Functions
### Func AllocatorLoggingWithLevel
```go
func AllocatorLoggingWithLevel(level int) []chromedp.ExecAllocatorOption
```
AllocatorOptsVerboseLogging provides ExecAllocator options for verbose
logging at the specified level.

### Func ChromeBinPathOnCI
```go
func ChromeBinPathOnCI() string
```
ChromeBinPathOnCI returns the Chrome binary path for CI.

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

### Func NewListenHandler
```go
func NewListenHandler[T any](ch chan<- T) func(ctx context.Context, ev any) bool
```
NewListenHandler returns a handler for a specific event type that forwards
the event to the provided channel.

### Func RunLoggingListener
```go
func RunLoggingListener(ctx context.Context, logger *slog.Logger, opts ...LoggingOption) chan struct{}
```
RunLoggingListener starts the logging listener for Chrome DevTools Protocol
events. It returns a channel that is closed when the goroutine that listens
on events terminates.

### Func SourceScript
```go
func SourceScript(ctx context.Context, script string) error
```
SourceScript loads a JavaScript script into the current page.

### Func UserDataDirOnCI
```go
func UserDataDirOnCI() string
```
UserDataDirOnCI returns the user data directory for Chrome on CI.

### Func WaitForPromise
```go
func WaitForPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams
```
WaitForPromise waits for a promise to resolve in the given evaluate
parameters.

### Func WithContextForCI
```go
func WithContextForCI(ctx context.Context, extraExecAllocOpts []chromedp.ExecAllocatorOption, opts ...chromedp.ContextOption) (context.Context, func())
```
WithContextForCI returns a chromedp context that may be different on a
CI system than when running locally. The CI configuration may disable
sandboxing etc. The ExecAllocator is always created with appropriate options
for the various CI environments and extraExecAllocOpts is appended to these.

### Func WithExecAllocatorForCI
```go
func WithExecAllocatorForCI(ctx context.Context, extraExecAllocOpts ...chromedp.ExecAllocatorOption) (context.Context, func())
```
WithExecAllocatorForCI returns a chromedp context with an ExecAllocator
configured appropriately for CI systems as opposed to when running locally.
The CI configuration may disable sandboxing for example.



## Types
### Type LoggingOption
```go
type LoggingOption func(*loggingOptions)
```
LoggingOption represents options to RunLoggingListener.

### Functions

```go
func WithAnyEventLogging() LoggingOption
```
WithAnyEventLogging enables logging for events of type 'any'. This is a
catch all and should generally be the last handler in the list.


```go
func WithConsoleLogging() LoggingOption
```
WithConsoleLogging enables logging of events of type
'runtime.EventConsoleAPICalled'.


```go
func WithEventEntryLogging() LoggingOption
```
WithEventEntryLogging enables logging of events of type
'log.EventEntryAdded'.


```go
func WithExceptionLogging() LoggingOption
```
WithExceptionLogging enables logging of events of type
'runtime.EventExceptionThrown'.


```go
func WithNetworkLogging() LoggingOption
```
WithNetworkLogging enables logging of events of type
'network.EventResponseReceived'.







