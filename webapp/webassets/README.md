# Package [cloudeng.io/webapp/webassets](https://pkg.go.dev/cloudeng.io/webapp/webassets?tab=doc)

```go
import cloudeng.io/webapp/webassets
```


## Functions
### Func NewAssets
```go
func NewAssets(prefix string, fsys fs.FS, opts ...AssetsOption) fs.FS
```
NewAssets returns an fs.FS that is configured to be optional reloaded from
the local filesystem or to be served directly from the supplied fs.FS.
The EnableReloading option is used to enable reloading. Prefix is prepended
to all names passed to the supplied fs.FS, which is typically obtained via
go:embed. See RelativeFS for more details.

### Func RelativeFS
```go
func RelativeFS(prefix string, fs fs.FS) fs.FS
```
RelativeFS wraps the supplied FS so that prefix is prepended to all of
the paths fetched from it. This is generally useful when working with
webservers where the FS containing files is created from 'assets/...' but
the URL path to access them is at the root. So /index.html can be mapped to
assets/index.html.

### Func ServeFile
```go
func ServeFile(wr io.Writer, fsys fs.FS, name string) (int, error)
```
ServeFile writes the specified file from the supplied fs.FS returning to the
supplied writer, returning an appropriate http status code.



## Types
### Type AssetsFlags
```go
type AssetsFlags struct {
	ReloadEnable    bool   `subcmd:"reload-enable,false,'if set, newer local filesystem versions of embedded asset files will be used'"`
	ReloadNew       bool   `subcmd:"reload-new-files,true,'if set, files that only exist on the local filesystem may be used'"`
	ReloadRoot      string `subcmd:"reload-root,$PWD,'the filesystem location that contains assets to be used in preference to embedded ones. This is generally set to the directory that the application was built in to allow for updated versions of the original embedded assets to be used. It defaults to the current directory. For external/production use this will generally refer to a different directory.'"`
	ReloadLogging   bool   `subcmd:"reload-logging,false,set to enable logging"`
	ReloadDebugging bool   `subcmd:"reload-debugging,false,set to enable debug logging"`
}
```
AssetsFlags represents the flags used to control loading of assets from the
local filesystem to override those original embedded in the application
binary.


### Type AssetsOption
```go
type AssetsOption func(a *assets)
```
AssetsOption represents an option to NewAssets.

### Functions

```go
func EnableDebugging() AssetsOption
```
EnableDebugging enables debug output.


```go
func EnableLogging() AssetsOption
```
EnableLogging enables logging using a built in logging function.


```go
func EnableReloading(location string, after time.Time, loadNew bool) AssetsOption
```
EnableReloading enables reloading of assets from the specified location
if they have changed since 'after'; loadNew controls whether new files,
ie. those that exist only in location, are loaded as opposed. See
cloudeng.io/io/reloadfs.


```go
func OptionsFromFlags(rf *AssetsFlags) []AssetsOption
```
OptionsFromFlags parses AssetsFlags to determine the options to be passed to
NewAssets()


```go
func UseLogger(logger func(action reloadfs.Action, name, path string, err error)) AssetsOption
```
UseLogger enables logging using the supplied logging function.







