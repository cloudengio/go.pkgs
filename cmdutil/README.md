# Package [cloudeng.io/cmdutil](https://pkg.go.dev/cloudeng.io/cmdutil?tab=doc)

```go
import cloudeng.io/cmdutil
```

Package cmdutil provides support for implementing command line utilities.

## Variables
### ErrInterrupt
```go
ErrInterrupt = errors.New("interrupted")

```
ErrInterrupt is returned as the cause for HandleInterrupt cancellations.



## Functions
### Func BuildInfoJSON
```go
func BuildInfoJSON() jsontext.Value
```
BuildInfoJSON returns the build information as a JSON raw message or nil if
the build information is not available.

### Func CopyAll
```go
func CopyAll(fromDir, toDir string, overwrite bool) error
```
CopyAll will create an exact copy, including permissions, of a local
filesystem hierarchy. The arguments must both refer to directories.
A trailing slash (/) for the fromDir copies the contents of fromDir rather
than fromDir itself. Thus:

    CopyAll("a/b", "c") is the same as CopyAll("a/b/", "c/b")
    and both create an exact copy of the tree a/b rooted at c/b.

If overwrite is set any existing files will be overwritten. Existing
directories will always have their contents updated. It uses os.Root scoped
APIs to prevent symlink TOCTOU traversal.

### Func CopyFile
```go
func CopyFile(from, to string, perms os.FileMode, overwrite bool) (returnErr error)
```
CopyFile will copy a local file with the option to overwrite an existing
file and to set the permissions on the new file. It uses chmod to explicitly
set permissions. It is not suitable for very large fles.

### Func Exit
```go
func Exit(format string, args ...any)
```
Exit formats and prints the supplied parameters to os.Stderr and then calls
os.Exit(1).

Deprecated: use Exitf instead.

### Func Exitf
```go
func Exitf(format string, args ...any)
```
Exitf formats and prints the supplied parameters to os.Stderr and then calls
os.Exit(1).

### Func HandleInterrupt
```go
func HandleInterrupt(ctx context.Context) (context.Context, context.CancelCauseFunc)
```
HandleInterrupt returns a context that is cancelled when an interrupt signal
is received. The returned CancelCauseFunc should be used to cancel the
context and will return ErrInterrupt as the cause.

### Func HandleSignals
```go
func HandleSignals(fn func(), signals ...os.Signal)
```
HandleSignals will asynchronously invoke the supplied function when the
specified signals are received.

### Func IsDir
```go
func IsDir(path string) bool
```
IsDir returns true iff path exists and is a directory.

### Func IsExplicitlySet
```go
func IsExplicitlySet(fs *flag.FlagSet, name string) bool
```
IsExplicitlySet returns true if the named flag was explicitly provided
on the command line (i.e. after FlagSet.Parse was called). It relies on
flag.FlagSet.Visit, which only visits flags that were set during parsing.

### Func ListDir
```go
func ListDir(dir string) ([]string, error)
```
ListDir returns the lexicographically ordered directories that lie beneath
dir. It uses os.Root scoped APIs to prevent symlink TOCTOU traversal.

### Func ListRegular
```go
func ListRegular(dir string) ([]string, error)
```
ListRegular returns the lexicographically ordered regular files that
lie beneath dir. It uses os.Root scoped APIs to prevent symlink TOCTOU
traversal.

### Func LogBuildInfo
```go
func LogBuildInfo(logger *slog.Logger)
```
LogBuildInfo logs build information using the provided logger.

### Func ReplaceAttrNoTime
```go
func ReplaceAttrNoTime(_ []string, a slog.Attr) slog.Attr
```
ReplaceAttrNoTime returns a slog.Attr with the time attribute removed.
This is useful for tests where the time is not deterministic.

### Func VCSInfo
```go
func VCSInfo() (goVersion, revision string, lastCommit, execModTime time.Time, dirty, ok bool)
```
VCSInfo extracts version control system information from the build info,
if available. It returns, in order:

  - goVersion: the version of Go the executable was built with.
  - revision: the vcs.revision recorded at build time.
  - lastCommit: the vcs.time recorded at build time, ie. the time of that
    revision.
  - execModTime: the modification time of the executable file. This is an
    approximation of when the executable was built and no more than that:
    it is the file's mtime, so it changes if the file is copied, touched or
    unpacked from an archive, and it is unrelated to the build info.
  - dirty: whether vcs.modified was recorded, ie. whether the tree had
    uncommitted changes when it was built.
  - ok: whether any vcs setting was found.

ok reports only on the vcs settings. goVersion and execModTime are
determined independently of them and may be set even when ok is false, as
happens for an executable built from a directory that is not a repository,
or with -buildvcs=false. The zero values of revision, lastCommit and dirty
are not distinguishable from values that were genuinely absent, so ok is
what should be tested before reporting them.

### Func WaitForExit
```go
func WaitForExit(ctx context.Context, funcs ...func() error) error
```
WaitForExit waits for all provided functions to return

### Func WaitForExitCtx
```go
func WaitForExitCtx(ctx context.Context, funcs ...func(context.Context) error) error
```
WaitForExitCtx is like WaitForExit but the functions are passed the context
that is cancelled when an error is returned by any of the functions.



## Types
### Type Logger
```go
type Logger struct {
	*slog.Logger
	// contains filtered or unexported fields
}
```
Logger represents a logger with an optional closer for the log file if one
is specified.

### Methods

```go
func (l *Logger) Close() error
```


```go
func (l *Logger) LogBuildInfo()
```
LogBuildInfo logs build information using the logger.




### Type LoggingConfig
```go
type LoggingConfig struct {
	Level      int    `yaml:"level" doc:"logging level: 0=error, 1=warn, 2=info, 3=debug"`
	File       string `yaml:"file" doc:"log file path. If not specified logs are written to stderr."`
	Format     string `yaml:"format" doc:"log format: text or json"`
	SourceCode bool   `yaml:"source_code" doc:"include source code file and line number in logs"`
}
```
LoggingConfig represents a logging configuration.

### Methods

```go
func (c LoggingConfig) Leveler() slog.Leveler
```


```go
func (c LoggingConfig) NewLogger(opts ...LoggingOption) (*Logger, error)
```
NewLogger creates a new logger based on the configuration.


```go
func (c LoggingConfig) NewLoggerMust(opts *slog.HandlerOptions, loggingOpts ...LoggingOption) *Logger
```
NewLoggerMust is like NewLogger but panics on error.


```go
func (c LoggingConfig) NewLoggerOpts(handlerOpts *slog.HandlerOptions, loggingOpts ...LoggingOption) (*Logger, error)
```
NewLoggerOpts creates a new logger based on the configuration and custom
handler options.


```go
func (c LoggingConfig) Options() *slog.HandlerOptions
```


```go
func (c LoggingConfig) WithFlagOverrides(fs *flag.FlagSet, lf LoggingFlags) LoggingConfig
```
WithFlagOverrides returns a new LoggingConfig with fields overridden by the
explicitly set flags in the provided FlagSet.




### Type LoggingFlags
```go
type LoggingFlags struct {
	Level      int    `subcmd:"log-level,0,'logging level: 0=error, 1=warn, 2=info, 3=debug'"`
	File       string `subcmd:"log-file,,'log file path. If not specified logs are written to stderr, if set to - logs are written to stdout'"`
	Format     string `subcmd:"log-format,json,'log format: text or json'"`
	SourceCode bool   `subcmd:"log-source-code,false,'include source code file and line number in logs'"`
}
```
LoggingFlags represents common logging related command line flags.

### Methods

```go
func (lf LoggingFlags) LoggingConfig() LoggingConfig
```
LoggingConfig returns the logging configuration represented by the flags.




### Type LoggingOption
```go
type LoggingOption func(*loggingOptions)
```
LoggingOption represents an option for configuring a logger beyond the
slog.HandlerOptions provided by the LoggingConfig.

### Functions

```go
func WithWriteCloser(wr io.WriteCloser) LoggingOption
```
WithWriteCloser sets the io.Writer for the slog.Logger to use and the Closer
to return. It is intended to allow for setting a writer that supports
log rotation or other logging destinations. If set, the File field of the
LoggingConfig is ignored and the WriteCloser passed here is used instead.






## Examples
### [ExampleLoggingFlags](https://pkg.go.dev/cloudeng.io/cmdutil?tab=doc#example-LoggingFlags)




