# Package [cloudeng.io/cicd](https://pkg.go.dev/cloudeng.io/cicd?tab=doc)

```go
import cloudeng.io/cicd
```

Package cicd provides support for working with CI environments.

## Constants
### LongRunningTestsEnv
```go
LongRunningTestsEnv = "CLOUDENG_LONG_RUNNING_TESTS"

```
LongRunningTestsEnv, is used to control whether and which long running tests
are to be run. No long running tests are run if this variable is not set.
If set, it may refer to either a numeric level (ie. 1, 2, etc) or a regular
expression as per go test '-run'. The numeric level is used to control ever
longer running tests, level 1 for < 10 minutes, level 2 for < 30 minutes,
etc. The regular expression is used to control which tests are run based on
their name, for example setting it to "Lifecycle" would run only tests with
"Lifecycle" in their name.



## Functions
### Func IsGitHubActions
```go
func IsGitHubActions() bool
```
IsGitHubActions returns true when running inside any GitHub Actions
workflow, regardless of whether the runner is hosted or self-hosted.

### Func IsGitHubHostedRunner
```go
func IsGitHubHostedRunner() bool
```
IsGitHubHostedRunner returns true when running on a GitHub-hosted runner.

### Func IsSelfHostedRunner
```go
func IsSelfHostedRunner() bool
```
IsSelfHostedRunner returns true when running on a self-hosted GitHub Actions
runner.

### Func LongRunningTest
```go
func LongRunningTest(t TestingT, level int)
```
LongRunningTest declares the calling test as a long-running one
of a given level that should only be run if requested via the
CLOUDENG_LONG_RUNNING_TESTS environment variable. See the documentation for
CLOUDENG_LONG_RUNNING_TESTS for details on how to control which long-running
tests are run. In short, if not set, no long running tests are run; if set
to a number, only long-running tests with that level and below are run;
if set to a non-number, only long-running tests with names matching that
regular expression are run.

### Func ParseLongRunningTestsEnv
```go
func ParseLongRunningTestsEnv() (enabled bool, level int, regex *regexp.Regexp, err error)
```
ParseLongRunningTestsEnv parses the CLOUDENG_LONG_RUNNING_TESTS environment
variable and returns whether long-running tests are enabled, the numeric
level if it is a number, and the regular expression if it is not a number.
The results are cached after the first call.

### Func SkipIf
```go
func SkipIf(t TestingTSkip, msg string, skipping bool)
```
SkipIf skips t if skipping is true, using msg as the skip message.

### Func SkipLinux
```go
func SkipLinux(t TestingTSkip)
```
SkipLinux skips t if running on Linux.

### Func SkipMacOS
```go
func SkipMacOS(t TestingTSkip)
```
SkipMacOS skips t if running on macOS.

### Func SkipWindows
```go
func SkipWindows(t TestingTSkip)
```
SkipWindows skips t if running on Windows.

### Func TestMain
```go
func TestMain[T TestingT](ctx context.Context, name string, w io.Writer, tests []func(T)) error
```
TestMain runs each test in tests with its own fresh *Testing. T must be
compatible with *Testing (i.e. *Testing or an interface it implements,
such as TestingT). Each test's name is derived from its function name via
reflection. Tests run in slice order. Output goes to w (nil → os.Stderr).



## Types
### Type ConfigManager
```go
type ConfigManager[T any] struct {
	// contains filtered or unexported fields
}
```
ConfigManager provides a means to manage configurations based on regex
patterns that can be matched against test names. It is useful for
centralizing the configuration of tests, especially those that are
externalized by a one package for use by multiple others. For example, when
an interface has multiple implementations for which tests can be shared.

### Methods

```go
func (c *ConfigManager[T]) Get(s string) T
```
Get returns the configuration associated with the first regex that matches
the input string. The regexes are evaluated in the order they were added via
Set. If no regex matches, the default configuration is returned, hence there
is no need to use a regex that matches all strings as the default case.


```go
func (c *ConfigManager[T]) Set(re *regexp.Regexp, config T)
```
Set associates a regex pattern with a specific configuration. It panics if
a nil regex is provided. The regexes are evaluated in the order they were
added via Set, so the first matching regex will determine the configuration
returned by Get.


```go
func (c *ConfigManager[T]) SetDefault(config T)
```
SetDefault sets the default configuration to be returned when no regex
matches.




### Type Testing
```go
type Testing struct {
	// contains filtered or unexported fields
}
```
Testing is a concrete implementation of TestingT for use outside the
test harness (e.g. integration tests run as binaries). Fatal/Fatalf and
Skip/Skipf terminate the current goroutine via runtime.Goexit, which runs
deferred functions before exiting — matching the behaviour of *testing.T.
Note that RunCleanups must be called to run registered cleanup functions
after the test body completes, matching *testing.T semantics.

### Functions

```go
func NewTesting(ctx context.Context, name string, w io.Writer) *Testing
```
NewTesting creates a Testing with the given name. Output goes to w; pass nil
to use os.Stderr.



### Methods

```go
func (t *Testing) Cleanup(f func())
```
Cleanup registers a function to be called when RunCleanups is invoked.
Functions are called in last-in-first-out order, matching *testing.T.


```go
func (t *Testing) Context() context.Context
```
Context returns the context for this test. The context is cancelled just
before RunCleanups is called, matching testing.T.Context() semantics.


```go
func (t *Testing) Error(args ...any)
```
Error marks the test as failed and writes a message.


```go
func (t *Testing) Errorf(format string, args ...any)
```
Errorf marks the test as failed and writes a formatted message.


```go
func (t *Testing) Failed() bool
```
Failed reports whether the test has been marked as failed.


```go
func (t *Testing) Fatal(args ...any)
```
Fatal marks the test as failed, writes a message, then terminates the
current goroutine via runtime.Goexit.


```go
func (t *Testing) Fatalf(format string, args ...any)
```
Fatalf marks the test as failed, writes a formatted message, then terminates
the current goroutine via runtime.Goexit.


```go
func (t *Testing) Helper()
```
Helper is a no-op; call-stack marking is not available outside the test
harness.


```go
func (t *Testing) Log(args ...any)
```
Log writes a message to the output writer.


```go
func (t *Testing) Logf(format string, args ...any)
```
Logf writes a formatted message to the output writer.


```go
func (t *Testing) Name() string
```
Name returns the name set at construction.


```go
func (t *Testing) Run(name string, f func(*Testing)) bool
```
Run mirrors testing.T.Run: it creates a child Testing named "parent/name",
runs f in a new goroutine (so Fatal/Skip only exit the child), waits for
completion. If the child fails, the parent is also marked as failed,
matching testing.T.Run semantics. Returns true if the child did not fail.


```go
func (t *Testing) RunCleanups()
```
RunCleanups runs all registered cleanup functions in LIFO order and clears
the cleanup list.


```go
func (t *Testing) Skip(args ...any)
```
Skip marks the test as skipped, writes a message, then terminates the
current goroutine via runtime.Goexit.


```go
func (t *Testing) Skipf(format string, args ...any)
```
Skipf marks the test as skipped, writes a formatted message, then terminates
the current goroutine via runtime.Goexit.


```go
func (t *Testing) Skipped() bool
```
Skipped reports whether the test has been marked as skipped.




### Type TestingT
```go
type TestingT interface {
	Helper()
	Context() context.Context
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
	Name() string
	Failed() bool
	Skipped() bool
	Log(args ...any)
	Logf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Skip(args ...any)
	Cleanup(f func())
}
```
TestingT mirrors testing.T and is implemented by cicd.Testing.


### Type TestingTSkip
```go
type TestingTSkip interface {
	Helper()
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
	Name() string
}
```




## Examples
### [ExampleConfigManager](https://pkg.go.dev/cloudeng.io/cicd?tab=doc#example-ConfigManager)
ExampleConfigManager demonstrates centralizing per-implementation test
configuration. A shared test suite calls Get with the running test name to
obtain the parameters that match that implementation; names that match no
regex fall back to the default.




