# Package [cloudeng.io/io/reloadfs](https://pkg.go.dev/cloudeng.io/io/reloadfs?tab=doc)

```go
import cloudeng.io/io/reloadfs
```

Package reloadfs provides an implemtation of fs.FS whose contents can be
selectively reloaded from disk. This allows for default contents to be
embedded in a binary, typically via go:embed, to be overridden at run time
if so desired. This can be useful for configuration files as well web server
assets.

## Functions
### Func New
```go
func New(root, prefix string, embedded fs.FS, opts ...ReloadableOption) fs.FS
```
New returns a new fs.FS that will dynamically reload files that have either
been changed, or optionally only exist, in the filesystem as compared to
the embedded files. See ReloadAfter and LoadNewFiles. If ReloadAfter is not
specified the current time is assumed, that is, files whose modification
time is after that will be reloaded. For a file to be reloaded either its
modification time or size have to differ. Comparing sizes can catch cases
where the file system time granularity is coarse. This leaves the one
corner case of a file being modified without changing either its size or
modification time.

The prefix is prepended to the argument supplied to Open to obtain the full
name passed to the supplied FS below. The root and prefix are prepended to
obtain the name to be used in the newly returned FS, typically a local file
system. For example, given:

    //go:embed assets/*.html
    var htmlAssets embed.FS

With the reloadable assets in /tmp/overrides, then New should be called as:

    New("/tmp/overrides", "assets", htmlAssets)

Currently files are reloaded when Open'ed, in the future support may be
provided to watch for changes and reload (or update metdata) those ahead of
time. Reloaded files are not cached and will be reloaded on every access.



## Types
### Type Action
```go
type Action int
```
Action represents the action taken by the implementation of fs.FS.

### Constants
### ReloadedExisting, ReloadedNewFile, Reused, NewFilesNotAllowed
```go
ReloadedExisting Action = iota
ReloadedNewFile
Reused
NewFilesNotAllowed

```
The set of available actions.



### Methods

```go
func (a Action) String() string
```




### Type ReloadableOption
```go
type ReloadableOption func(*reloadable)
```
ReloadableOption represents an option to ReloadableFS.

### Functions

```go
func DebugOutput(enable bool) ReloadableOption
```
DebugOutput debug output.


```go
func LoadNewFiles(a bool) ReloadableOption
```
LoadNewFiles controls whether files that exist only in file system and not
in the embedded FS are returned. If false, only files that exist in the
embedded FS may be reloaded from the new FS.


```go
func ReloadAfter(t time.Time) ReloadableOption
```
ReloadAfter sets the time after which assets are to be reloaded rather than
reused. Note that the current implementation of go:embed does not record


```go
func UseLogger(logger func(action Action, name, path string, err error)) ReloadableOption
```
UseLogger provides a logger to be used by the underlying implementation.







