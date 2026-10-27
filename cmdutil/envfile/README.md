# Package [cloudeng.io/cmdutil/envfile](https://pkg.go.dev/cloudeng.io/cmdutil/envfile?tab=doc)

```go
import cloudeng.io/cmdutil/envfile
```

Package envfile parses files containing shell-style environment variable
definitions of the form NAME=VALUE, as commonly used in .env files.

## Functions
### Func ChainMapping
```go
func ChainMapping(mappings ...func(string) string) func(string) string
```
ChainMapping returns a mapping function that calls each of the supplied
mapping functions in turn and returns the first non-empty value any of
them yields, hence earlier mappings take precedence over later ones. It is
typically used to fall back from an envfile to the process environment,
or to layer a local envfile over shared defaults:

    os.Expand(s, envfile.ChainMapping(local.Mapping, shared.Mapping, os.Getenv))

Note that a mapping which returns an empty value is indistinguishable from
one that has no value for the name, so a variable defined as empty by an
earlier mapping does not prevent a later one from supplying a value for it.

Nil mappings are skipped, so an absent source can be passed as nil. With no
mappings, or with none that yield a value, the returned function returns ""
for every name; it is never nil and is always safe to call.

### Func DefaultsMapping
```go
func DefaultsMapping(defaults map[string]string) func(string) string
```
DefaultsMapping returns a mapping, with the signature expected by os.Expand,
that takes its values from defaults, which it copies. It is intended to come
after the other mappings in a chain, see ChainMapping, so that it supplies a
value only for a variable that none of them defines:

    mapping := envfile.ChainMapping(
    	envfile.ExpandingMapping(vars),
    	envfile.DefaultsMapping(map[string]string{"REGION": "us-west-2"}),
    	unresolved.Mapping)

As with any mapping, a default that is empty is indistinguishable from none.

### Func ExpandingMapping
```go
func ExpandingMapping(vars *Variables) func(string) string
```
ExpandingMapping returns a mapping, with the signature expected by
os.Expand, that takes its values from the process environment and,
for any variable that the environment does not define, from vars.
The shell environment overrides the definitions, so that a value configured
for everyday use can be changed for a single invocation without editing the
file. Note that a variable set to an empty value is indistinguishable from
one that is not set, so an empty value in the environment does not hide a
definition in vars.

A value taken from vars may itself contain references, as in

    PROFILE=$HOME/Downloads/example.provisionprofile

and those are expanded too, from the environment first and then from vars,
since Parse does not expand them when it reads a file and a single pass over
a configuration file would otherwise leave the $HOME above in the value. A
reference that resolves to nothing is left as ${NAME}, so that it is visible
rather than silently empty, and expansion is bounded so that a definition
referring to itself terminates.

Values taken from the environment are used as they are: the shell has
already expanded them, and a value that merely contains a '$', such as a
password, must not be altered.

A nil vars yields the process environment alone.

### Func Parse
```go
func Parse(r io.Reader) (map[string]string, error)
```
Parse reads environment variable definitions from r and returns a map of
variable names to their values. Each non-blank, non-comment line must be of
the form:

    [export] NAME=VALUE

VALUE may be:
  - Unquoted: terminated by an unquoted # (inline comment) or end of line;
    leading and trailing whitespace is trimmed.
  - Single-quoted ('VALUE'): literal content, no escape processing.
  - Double-quoted ("VALUE"): backslash escapes \n \t \r \" \\ \$ are
    interpreted; other \X sequences are passed through as-is.

Lines whose first non-whitespace character is # are comments and are
skipped. The export keyword prefix is accepted and ignored. Lines without
an = are silently skipped. If a name appears more than once the last value
wins.

### Func ParseFile
```go
func ParseFile(filename string) (map[string]string, error)
```
ParseFile is a convenience wrapper around Parse that opens and reads a file.



## Types
### Type StructEnv
```go
type StructEnv struct {
	// contains filtered or unexported fields
}
```
StructEnv expands environment variable references in struct fields using the
`use_env` and `use_env_file` struct tags. It caches parsed envfiles across
multiple calls to Expand so each file is read at most once. StructEnv is
safe for concurrent use.

### Methods

```go
func (se *StructEnv) Expand(s any) error
```
Expand processes the exported string fields of the struct pointed to by s
and expands environment variable references in those fields.

Two struct tags are recognised:

  - `use_env`: the field's current value may contain $VAR or ${VAR}
    references that are expanded using the process environment
    (os.LookupEnv).

  - `use_env_file`: the field's current value encodes both the source file
    and the variable reference in the form:

    filename:$VAR filename:${VAR}

    The filename is parsed greedily: it is everything before the last ':'
    that is immediately followed by '$'. This allows filenames that contain
    ':' (e.g. absolute Windows paths). The file is parsed with ParseFile and
    the variable reference is expanded using the resulting map. If the field
    value does not match the pattern it is left unchanged.

Both tags use the same ${VAR} / $VAR syntax as os.Expand. Non-string fields
are silently skipped regardless of tags.

The struct must be passed as a non-nil pointer.




### Type Unresolved
```go
type Unresolved struct {
	// contains filtered or unexported fields
}
```
Unresolved records the variables that could not be expanded. Its Mapping
method is intended to be the last in a chain of mappings, see ChainMapping,
so that it is called only for a variable that none of the others defines,
which it records, and resolves to nothing. Once the expansion is done,
Err reports whether any were recorded, and Names returns them. The zero
value is ready for use, and it is safe for concurrent use.

### Methods

```go
func (u *Unresolved) Err() error
```
Err returns nil if no variable was recorded, or an error that contains an
*UnresolvedError for each that was, see errors.M.


```go
func (u *Unresolved) Mapping(name string) string
```
Mapping records name, once, however often it is called for it, and returns
"".


```go
func (u *Unresolved) Names() []string
```
Names returns the names of the variables recorded, in the order in which
they were first recorded.




### Type UnresolvedError
```go
type UnresolvedError struct {
	Name string
}
```
UnresolvedError is the error recorded by Unresolved for a variable that
could not be expanded.

### Methods

```go
func (e *UnresolvedError) Error() string
```
Error implements error.




### Type Variables
```go
type Variables struct {
	// contains filtered or unexported fields
}
```
Variables holds the environment variable definitions read from one or more
envfiles. Its Mapping method has the signature expected for the mapping
argument of os.Expand, and of the many APIs that accept such a function,
so the definitions in an envfile can be used to expand $VAR and ${VAR}
references anywhere one is called for:

    os.Expand(s, vars.Mapping)

Note that Parse, and hence Read, does not itself expand references that
appear within an envfile's own values; use Expand, or Mapping, to resolve
those.

The read-only methods are safe to call on a nil or zero-value Variables,
which behaves as one with no definitions. Merge and MergeFile require a
non-nil receiver.

### Functions

```go
func NewVariables(vars map[string]string) *Variables
```
NewVariables returns a Variables containing a copy of vars.


```go
func Read(r io.Reader) (*Variables, error)
```
Read reads environment variable definitions from r, as parsed by Parse,
and returns them as a Variables.


```go
func ReadFile(filename string) (*Variables, error)
```
ReadFile is a convenience wrapper around Read that opens and reads a file.



### Methods

```go
func (v *Variables) All() map[string]string
```
All returns a copy of the definitions held by v.


```go
func (v *Variables) Expand(s string) string
```
Expand is shorthand for os.Expand(s, v.Mapping) and hence replaces
references to undefined variables with "" and expands s in a single pass,
leaving any references in the values it substitutes unexpanded.


```go
func (v *Variables) Len() int
```
Len returns the number of variables defined.


```go
func (v *Variables) Lookup(name string) (string, bool)
```
Lookup returns the value defined for name and whether it is defined.


```go
func (v *Variables) Mapping(name string) string
```
Mapping returns the value defined for name, or "" if name is not defined.
Its signature matches the mapping argument of os.Expand. Use Lookup to
distinguish a variable defined as empty from one that is not defined.


```go
func (v *Variables) Merge(r io.Reader) error
```
Merge reads further definitions from r and merges them into v, with the
values read from r overwriting any already defined for the same name.
If r cannot be parsed v is left unchanged.


```go
func (v *Variables) MergeFile(filename string) error
```
MergeFile is a convenience wrapper around Merge that opens and reads a file.






## Examples
### [ExampleChainMapping](https://pkg.go.dev/cloudeng.io/cmdutil/envfile?tab=doc#example-ChainMapping)




