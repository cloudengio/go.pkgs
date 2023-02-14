# Package [cloudeng.io/cmdutil](https://pkg.go.dev/cloudeng.io/cmdutil?tab=doc)

```go
import cloudeng.io/cmdutil
```

Package cmdutil provides support for implementing command line utilities.

## Functions
### Func CopyAll
```go
func CopyAll(fromDir, toDir string, ovewrite bool) error
```
CopyAll will create an exact copy, including permissions, of a local
filesystem hierarchy. The arguments must both refer to directories.
A trailing slash (/) for the fromDir copies the contents of fromDir rather
than fromDir itself. Thus:

    CopyAll("a/b", "c") is the same as CopyAll("a/b/", "c/b")
    and both create an exact copy of the tree a/b rooted at c/b.

If overwrite is set any existing files will be overwritten. Existing
directories will always have their contents updated. It is suitable for very
large directory trees since it uses filepath.Walk.

### Func CopyFile
```go
func CopyFile(from, to string, perms os.FileMode, overwrite bool) (returnErr error)
```
CopyFile will copy a local file with the option to overwrite an existing
file and to set the permissions on the new file. It uses chmod to explicitly
set permissions. It is not suitable for very large fles.

### Func Exit
```go
func Exit(format string, args ...interface{})
```
Exit formats and prints the supplied parameters to os.Stderr and then calls
os.Exit(1).

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

### Func ListDir
```go
func ListDir(dir string) ([]string, error)
```
ListDir returns the lexicographically ordered directories that lie beneath
dir.

### Func ListRegular
```go
func ListRegular(dir string) ([]string, error)
```
ListRegular returns the lexicographically ordered regular files that lie
beneath dir.

### Func ParseYAMLConfig
```go
func ParseYAMLConfig(spec []byte, cfg interface{}) error
```
ParseYAMLConfig will parse the yaml config in spec into the requested type.
It provides improved error reporting via YAMLErrorWithSource.

### Func ParseYAMLConfigFile
```go
func ParseYAMLConfigFile(file string, cfg interface{}) error
```
ParseYAMLConfigFile reads a yaml config file as per ParseYAMLConfig.

### Func ParseYAMLConfigString
```go
func ParseYAMLConfigString(spec string, cfg interface{}) error
```
ParseYAMLConfigString is like ParseYAMLConfig but for a string.

### Func YAMLErrorWithSource
```go
func YAMLErrorWithSource(spec []byte, err error) error
```
YAMLErrorWithSource returns an error that includes the yaml source code that
was the cause of the error to help with debugging YAML errors. Note that the
errors reported for the yaml parser may be inaccurate in terms of the lines
the error is reported on. This seems to be particularly true for lists where
errors with use of tabs to indent are often reported against the previous
line rather than the offending one.




