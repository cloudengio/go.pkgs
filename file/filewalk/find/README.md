# Package [cloudeng.io/file/filewalk/find](https://pkg.go.dev/cloudeng.io/file/filewalk/find?tab=doc)

```go
import cloudeng.io/file/filewalk/find
```

Package find provides a filewalk.Handler that can be used to locate
prefixes/directories and files based on file.Matcher expressions.

## Functions
### Func NeedsStat
```go
func NeedsStat(prefixMatcher, fileMatcher matcher.T) bool
```
NeedsStat determines if either of the supplied matcher.T's include operands
that would require a call to fs.Stat or fs.Lstat.

### Func New
```go
func New(fs filewalk.FS, ch chan<- Found, opts ...Option) filewalk.Handler[struct{}]
```
New returns a filewalk.Handler that can match on prefix/directory names
as well as filenames using file.Matcher expressions. The prefixMatcher is
applied to the prefix/directory and if prune is true no further processing
of that directory will take place. The fileMatcher is applied to the
filename (without its parent).

### Func Parse
```go
func Parse(input string) (matcher.T, error)
```
Parse parses the supplied input into a matcher.T. The supported syntax
is a boolean expression with and (&&), or (||) and grouping, via ().
The supported operands are:

    	name='glob-pattern'
    	iname='glob-pattern'
    	re='regexp'
    	type='f|d|l'
    	newer='date' in time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly

     Note that the single quotes are optional unless a white space is present
     in the pattern.



## Types
### Type Found
```go
type Found struct {
	Prefix string
	Name   string
	Err    error
}
```
Found is used to send matches or errors to the client.


### Type Option
```go
type Option func(*options)
```
Option represents an option for New.

### Functions

```go
func WithFileMatcher(m matcher.T) Option
```
WithFileMatcher specifies the matcher.T to use for matching filenames.
If none is supplied then no matches will be returned. The matcher.T is
applied to name of the entry within a prefix/directory.


```go
func WithFollowSoftlinks(v bool) Option
```
WithFollowSoftlinks specifies that the filewalk.Handler should follow
softlinks by calling fs.Stat rather than the default of calling fs.Lstat.


```go
func WithPrefixMatcher(m matcher.T) Option
```
WithPrefixMatcher specifies the matcher.T to use for matching
prefixes/directories. If none is supplied then no matches will be returned.
The matcher.T is applied to the full path of the prefix/directory.


```go
func WithPrune(v bool) Option
```
WithPrune specifies that the filewalk.Handler should prune directories
that match the prefixMatcher. That is, once a directory is matched no
subdirectories will be examined.


```go
func WithStat(v bool) Option
```
WithStat specifies that the filewalk.Handler should call fs.Stat or fs.Lstat
for files. Note that stat is always called for directories.







