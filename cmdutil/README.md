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

### Func ConfigFilePath
```go
func ConfigFilePath(appDir string) string
```
ConfigFilePath returns the search path for configuration files for the
application directory appDir, in the form expected by filepath.SplitList and
hence by LocateConfigFilePath. It contains, in order:

 1. the current directory
 2. appDir within the user's configuration directory as reported by
    os.UserConfigDir, ie. $HOME/Library/Application Support/appDir on macOS,
    $XDG_CONFIG_HOME/appDir or $HOME/.config/appDir elsewhere on unix,
    and %AppData%\appDir on windows
 3. on unix, including macOS, $HOME/.config/appDir

Every directory in the path belongs to the user: no machine-wide location,
such as /etc/appDir, is searched. Duplicates are omitted, so on a unix
system where os.UserConfigDir is itself $HOME/.config the directory appears
once rather than twice.

A directory is included whether or not it exists; locating a file in one
that does not is simply a miss.

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

### Func IsCleanShutdown
```go
func IsCleanShutdown(err error, cancelIsEnough bool, clean ...error) bool
```
IsCleanShutdown reports whether err is the result of a clean, user-initiated
shutdown, ie. one triggered by a signal handled via HandleInterrupt or
HandleSignals, or by cancelling the command's context, rather than by a
failure. It is intended for the top level of a CLI, to decide between
exiting quietly and reporting an error:

    if err := cli.Dispatch(ctx); err != nil {
    	if cmdutil.IsCleanShutdown(err, false) {
    		return
    	}
    	cmdutil.Exitf("error: %v", err)
    }

context.Canceled and ErrInterrupt always count as clean, as does any
error supplied in clean, which lets a command add its own benign shutdown
sentinels, eg. http.ErrServerClosed or a client's own 'closed' error.

cancelIsEnough selects between the two readings of an error that aggregates
several others. When true, err is clean if any error in its tree is clean,
which suits a command whose shutdown path cannot report anything but a
cancellation. When false, every error in the tree must be clean. The latter
is what a command that collects errors across concurrent components needs:
such a command typically aggregates ctx.Err() alongside whatever its
components returned, so a run that was both cancelled and genuinely broken
holds context.Canceled next to the real failure, and reporting it as clean
would exit quietly on a real error.

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

### Func LocateConfigFileInDir
```go
func LocateConfigFileInDir(name, appDir string, suffixes ...string) string
```
LocateConfigFileInDir searches for a configuration file with the
given name using the search path returned by ConfigFilePath, ie. it is
LocateConfigFilePath applied to that path. If name is an absolute path it is
used directly. If no matching file is found an empty string is returned.

### Func LocateConfigFilePath
```go
func LocateConfigFilePath(name, path string, suffixes ...string) string
```
LocateConfigFilePath searches for the first configuration file that matches
the given name in path (as expected by filepath.SplitList). The suffixes
are the filename suffixes to try, in order of preference, and include any
leading '.', eg. ".yaml", ".yml". The search order is as follows:
 1. If name is an absolute path and the file exists, it is returned, if it
    does not exist an empty string is returned. If it exists and an error
    other than "file not found" occurs, the filename is returned so that the
    caller can attempt to open it and handle the error.
 2. For each directory in path, the following are checked in order: a.
    dir/name<suffix> for each suffix b. dir/.name<suffix> for each suffix

3. If no matching file is found, an empty string is returned.

If name already ends in one of the suffixes then only that suffix is used,
so that dir/name.yaml is never tested for as dir/name.yaml.yml.
If no suffixes are supplied then name is used as is, ie. only dir/name and
dir/.name are tested for.

The suffixes play no part when name is an absolute path: such a name is used
exactly as given, so /dir/name is never tested for as /dir/name.yaml.

### Func LocateConfigFilesInDir
```go
func LocateConfigFilesInDir(appDir string, suffixes []string, names ...string) (files, missing []string)
```
LocateConfigFilesInDir searches for each of the named configuration files
using the search path returned by ConfigFilePath, resolving each name
exactly as LocateConfigFileInDir resolves it. It returns the files that were
found, in the order their names were supplied, and the names that were not
found.

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
### [ExampleIsCleanShutdown](https://pkg.go.dev/cloudeng.io/cmdutil?tab=doc#example-IsCleanShutdown)
ExampleIsCleanShutdown illustrates the decision a CLI makes at its top level
once its context has been cancelled, and the difference the cancelIsEnough
flag makes to a command that aggregates errors from several components.

### [ExampleLoggingFlags](https://pkg.go.dev/cloudeng.io/cmdutil?tab=doc#example-LoggingFlags)




