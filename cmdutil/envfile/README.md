# Package [cloudeng.io/cmdutil/envfile](https://pkg.go.dev/cloudeng.io/cmdutil/envfile?tab=doc)

```go
import cloudeng.io/cmdutil/envfile
```

Package envfile parses files containing shell-style environment variable
definitions of the form NAME=VALUE, as commonly used in .env files.

## Functions
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
`env` and `envfile` struct tags. It caches parsed envfiles across multiple
calls to Expand so each file is read at most once.

### Methods

```go
func (se *StructEnv) Expand(s any) error
```
Expand processes the exported string fields of the struct pointed to by s
and expands environment variable references in those fields.

Two struct tags are recognised:

  - `env`: the field's current value may contain $VAR or ${VAR} references
    that are expanded using the process environment (os.LookupEnv).

  - `envfile:"filename"`: the named file is parsed with ParseFile and $VAR
    or ${VAR} references in the field's current value are expanded using the
    variables defined in that file.

Both tags use the same ${VAR} / $VAR syntax as os.Expand. A field value that
contains no $ is treated as a literal and left unchanged. Non-string fields
are silently skipped regardless of tags.

The struct must be passed as a non-nil pointer.







