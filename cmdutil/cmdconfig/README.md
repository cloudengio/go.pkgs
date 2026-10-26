# Package [cloudeng.io/cmdutil/cmdconfig](https://pkg.go.dev/cloudeng.io/cmdutil/cmdconfig?tab=doc)

```go
import cloudeng.io/cmdutil/cmdconfig
```


## Constants
### ConfigExtensionName
```go
ConfigExtensionName = "config"

```
ConfigExtensionName is the name of the extension returned by
ConfigExtension, for use in the YAML template of a command tree:

    {{range subcmdExtension "config"}}{{.}}
    {{end}}



## Functions
### Func ConfigExtension
```go
func ConfigExtension[T any](cc ConfigCommands[T]) subcmd.Extension
```
ConfigExtension returns an Extension that adds the 'config show' and 'config
describe' commands, implemented by cc, to a command tree. It is used with
ConfigHook, which makes the configuration available to Show.

### Func LoadFiles
```go
func LoadFiles(ctx context.Context, cfg any, loc *Located, opts ...cmdyaml.Option) error
```
LoadFiles is like Load for files that have already been located. Callers can
add to or edit the files in loc first, for example to add a YAML file that
is not named by a FilesFlag.

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

### Func SearchPathFor
```go
func SearchPathFor(userDir, xdgDir string) string
```
SearchPathFor is like SearchPath for an application that uses a different
directory name in the user's configuration directory, userDir, than in
$HOME/.config, xdgDir.



## Types
### Type ConfigCommands
```go
type ConfigCommands[T any] struct {
	// Summary is written at the top of the output of Describe.
	Summary string
	// Describers are called by Describe, in order, after the documentation
	// for T, to write further documentation to the output.
	Describers []func(io.Writer) error
	// Output is where the output is written, os.Stdout if nil.
	Output io.Writer
}
```
ConfigCommands implements commands to show, and to document, the YAML
configuration of type T. The methods are subcmd.Runners; see ConfigExtension
for adding them to a command set.

### Methods

```go
func (c ConfigCommands[T]) Describe(_ context.Context, _ any, _ []string) error
```
Describe writes the documentation for the configuration type T, from its
'doc' struct tags, see structdoc.Describe, followed by whatever Describers
write.


```go
func (c ConfigCommands[T]) Show(ctx context.Context, _ any, _ []string) error
```
Show writes the configuration of type T carried by ctx, using
cmdutil.ContextWithValue, as YAML preceded by comments naming the files that
it was read from if the context carries the *Located for them.




### Type ConfigHook
```go
type ConfigHook[T any] struct {
	// ID identifies the hook in error reporting, it defaults to
	// "withConfigPrehook".
	ID string
	// Files is the flag that holds the names of the files.
	Files *FilesFlag
	// Spec determines how the files are located.
	Spec Spec
	// Initial is the value of the configuration before the files are parsed
	// into it, and the value that is used if there are none.
	Initial T
	// Required causes it to be an error for there to be no YAML file.
	Required bool
	// ParseOptions are passed to Load.
	ParseOptions []cmdyaml.Option
	// ValidateFields causes cmdyaml.ValidateFields to be called, before the
	// Validate method of the configuration, if it has one.
	ValidateFields bool
	// Store, if not nil, is called with the configuration, and the files it was
	// located in, to add anything else the commands need to the context. The
	// configuration, as a T, and the files, as a *Located, are always
	// added, see cmdutil.ContextWithValue.
	Store func(ctx context.Context, cfg T, loc *Located) context.Context
}
```
ConfigHook is a PreHook, see its Run method, that loads the configuration of
a command from the files named by a FilesFlag, registered as a global flag,
and the environment variables files that go with them.

### Methods

```go
func (h ConfigHook[T]) Run(ctx context.Context) (context.Context, string, subcmd.PostHook, error)
```
Run locates and parses the files, as Load does, and validates the
configuration if it was read from a file: the fields, if ValidateFields
is set, and then the configuration itself if it has a Validate method.
It adds the configuration and the files to the context and calls Store.
Errors from parsing and validating say which files they are in and that they
need to be edited. It implements PreHook.




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

### Functions

```go
func Load(ctx context.Context, cfg any, ff *FilesFlag, spec Spec, opts ...cmdyaml.Option) (*Located, error)
```
Load locates the files held by ff, as per spec, and parses the YAML files
among them into cfg. Parsing is strict, see WithStrictFields, and $VAR and
${VAR} references in the YAML are expanded from the process environment and,
for any variable that it does not define, from the environment variables
files, see envfile.ExpandingMapping; where several define a variable the
later file takes precedence. opts are applied after, and so can override,
those. The files that were located are returned, even if there was an error
parsing them, so that they can be reported, but not if there was an error
locating them.

Once any files have been parsed, or if there are none, the SetDefaults
method of cfg, if it has one, is called, so that it can fill in whatever
the files did not set, whether or not there were any. It is not an error for
there to be no YAML file: a caller for which one is required should check
the result.



### Methods

```go
func (l *Located) Files(typ FileType) []string
```
Files returns the files of the specified type.




### Type Spec
```go
type Spec struct {
	// Path is the search path for the files, in the form expected by
	// filepath.SplitList, see SearchPath and SearchPathFor. It defaults to the
	// current directory.
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





