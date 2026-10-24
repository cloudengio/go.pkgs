# Package [cloudeng.io/cmdutil/cmdconfig](https://pkg.go.dev/cloudeng.io/cmdutil/cmdconfig?tab=doc)

```go
import cloudeng.io/cmdutil/cmdconfig
```


## Functions
### Func LocateFile
```go
func LocateFile(name, appDir string, suffixes ...string) string
```
LocateFile searches for a configuration file with the given name using the
search path returned by SearchPath, ie. it is LocateFilePath applied to that
path. If name is an absolute path it is used directly. If no matching file
is found an empty string is returned.

### Func LocateFilePath
```go
func LocateFilePath(name, path string, suffixes ...string) string
```
LocateFilePath searches for the first configuration file that matches the
given name in path (as expected by filepath.SplitList). The suffixes are the
filename suffixes to try, in order of preference, and include any leading
'.', eg. ".yaml", ".yml". The search order is as follows:
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

### Func LocateFiles
```go
func LocateFiles(appDir string, suffixes []string, names ...string) (files, missing []string)
```
LocateFiles searches for each of the named configuration files using
the search path returned by SearchPath, resolving each name exactly as
LocateFile resolves it. It returns the files that were found, in the order
their names were supplied, and the names that were not found.

### Func LocateFilesPath
```go
func LocateFilesPath(path string, suffixes []string, names ...string) (files, missing []string)
```
LocateFilesPath searches for each of the named configuration files using the
specified search path and suffixes. It returns the files that were found,
in the order their names were supplied, and the names that were not found.

### Func SearchPath
```go
func SearchPath(appDir string) string
```
SearchPath returns the search path for configuration files for the
application directory appDir, in the form expected by filepath.SplitList and
hence by LocateFilePath. It contains, in order:

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



## Types
### Type FileType
```go
type FileType int
```
FileType represents the type of configuration file: YAML, .env, or JSON.

### Constants
### FileTypeYAML, FileTypeEnv, FileTypeJSON
```go
FileTypeYAML FileType = iota
FileTypeEnv
FileTypeJSON

```



### Methods

```go
func (t FileType) String() string
```
String implements fmt.Stringer.




### Type FilesFlag
```go
type FilesFlag struct {
	// contains filtered or unexported fields
}
```
FilesFlag represents a command-line flag that can hold multiple YAML, json
and .env files. It accepts comma-separated values and can also be repeated.
The type of each file is determined by its suffix. Files that are set,
on the command line, replace the defaults, see SetDefaults, and are located
using Locate.

### Methods

```go
func (f *FilesFlag) DefaultValues() string
```
DefaultValues returns a comma-separated string of all default file paths for
YAML, .env, and JSON files.


```go
func (f *FilesFlag) Files(typ FileType, ignoreDefaults bool) []string
```
Files returns a copy of the list of files of the specified type set in the
flag or the default files if none were set. Use HasSet to check if any were
explicitly set.


```go
func (f *FilesFlag) HasSet(typ FileType) bool
```
HasSet returns true if any files of the specified type were explicitly set
in the flag.


```go
func (f *FilesFlag) Locate(spec Spec) (*Located, error)
```
Locate locates the files of each type held by the flag, as per spec.
The files of a type are, in order of precedence, those set in the flag,
those named by the environment variable for that type, and the defaults.
Files that are set, in the flag or by the environment variable, must exist
and an error is returned for any that does not. Defaults are optional:
the ones that do not exist are omitted, and it is for the caller to decide
whether it is an error for there to be none.


```go
func (f *FilesFlag) LocateFilesPath(typ FileType, ignoreDefault bool, path string) ([]string, []string)
```
LocateFilesPath locates the files of the specified type, in path, returning
those that were found and the names of those that were not. The files are
those set in the flag, or the defaults if none were set unless ignoreDefault
is true.


```go
func (f *FilesFlag) Set(vals string) error
```
Set parses a comma-separated list of file paths and categorizes them into
YAML, .env, and JSON files based on their suffixes. The flag may be repeated
on the command line.


```go
func (f *FilesFlag) SetDefault(vals string) error
```
SetDefault replaces the default files with the comma-separated
list of file paths, categorizing them as Set does. It is called by
flags.RegisterFlagsInStruct to set the default from the struct tag or from
DefaultValues, and so is idempotent.


```go
func (f *FilesFlag) SetDefaults(files ...string)
```
SetDefaults adds comma-separated lists of file paths, categorized as Set
does, to the default files. Files with an unsupported suffix are ignored.


```go
func (f *FilesFlag) String() string
```
String implements flag.Value, and returns the files that were set.




### Type Located
```go
type Located struct {
	YAML, Env, JSON []string
	// SearchPath is the path that the files were searched for in.
	SearchPath string
}
```
Located is the result of locating the files of a FilesFlag.

### Methods

```go
func (l *Located) Files(typ FileType) []string
```
Files returns the files of the specified type.




### Type Spec
```go
type Spec struct {
	// AppDir is the application directory that the files are searched for
	// in, see SearchPath.
	AppDir string
	// Path, if set, is the search path, in the form expected by
	// filepath.SplitList, to use instead of that derived from AppDir.
	Path string
	// EnvVars names, per type of file, an environment variable whose value is
	// a comma-separated list of files, which are used in place of the defaults
	// if no files of that type were set in the flag. Files named by the
	// environment variable are treated as having been set.
	EnvVars map[FileType]string
	// FirstOnly lists the types of file for which at most one file is
	// wanted: the first of those named that is found. Otherwise all of
	// those named that are found are returned, in the order named.
	FirstOnly []FileType
}
```
Spec specifies how the files of a FilesFlag are located.





