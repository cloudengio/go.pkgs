# Package [cloudeng.io/cicd](https://pkg.go.dev/cloudeng.io/cicd?tab=doc)

```go
import cloudeng.io/cicd
```

Package ci provides support for working with CI environments.

## Constants
### LongRunningTestsEnv
```go
LongRunningTestsEnv = "CLOUDENG_LONG_RUNNING_TESTS"

```
LongRunningTestsEnv, is used to control whether and which long running tests
are to be run. No long running tests are run if this variable is not set. If
set, it may refer to either a numeric level (ie. 1, 2, etc) or a a regular
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

### Func LongRunnningTest
```go
func LongRunnningTest(t TestingT, level int)
```
LongRunningTest declares the calling test as a long-running one
of a given leve that should only be run if requested via the
CLOUDENG_LONG_RUNNING_TESTS environment variable. See the documentation for
CLOUDENG_LONG_RUNNING_TESTS for details on how to control which long-running
tests are run. In short, if not set, no long running tests are run; if set
to a number, only long-running tests with that level and above are run;
if set to a non-number, only long-running tests with names matching that
regular expression are run.

### Func ParseLongRunningTestsEnv
```go
func ParseLongRunningTestsEnv() (enabled bool, level int, regex *regexp.Regexp)
```
ParseLongRunningTestsEnv parses the CLOUDENG_LONG_RUNNING_TESTS environment
variable and returns whether long-running tests are enabled, the numeric
level if it is a number, and the regular expression if it is not a number.
The results are cached after the first call.

### Func SkipIf
```go
func SkipIf(t TestingT, msg string, skipping bool)
```
SkipIf skips t if skipping is true, using msg as the skip message.

### Func SkipLinux
```go
func SkipLinux(t TestingT)
```
SkipLinux skips t if running on Linux.

### Func SkipMacOS
```go
func SkipMacOS(t TestingT)
```
SkipMacOS skips t if running on macOS.

### Func SkipWindows
```go
func SkipWindows(t TestingT)
```
SkipWindows skips t if running on Windows.



## Types
### Type TestingT
```go
type TestingT interface {
	Helper()
	Skipf(format string, args ...any)
	Name() string
}
```





