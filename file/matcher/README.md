# Package [cloudeng.io/file/matcher](https://pkg.go.dev/cloudeng.io/file/matcher?tab=doc)

```go
import cloudeng.io/file/matcher
```


## Functions
### Func DirSize
```go
func DirSize(opname, value string, larger bool) boolexpr.Operand
```
DirSize returns a 'directory size' operand. The value is not validated until
a matcher.T is created using New. The size must be expressed as an integer.
If larger is true then the comparison is performed using >=, otherwise <.
The operand requires that the value being matched implements FileTypeIfc and
DirSizeIfc.

### Func DirSizeLarger
```go
func DirSizeLarger(n, v string) boolexpr.Operand
```

### Func DirSizeSmaller
```go
func DirSizeSmaller(n, v string) boolexpr.Operand
```

### Func FileSize
```go
func FileSize(opname, value string, larger bool) boolexpr.Operand
```
FileSize returns a 'file size' operand. The value is not validated until a
matcher.T is created using New. The size may be expressed as an in binary
(GiB, KiB) or decimal (GB, KB) or as bytes (eg. 1.1GB, 1GiB or 1000).
If larger is true then the comparison is performed using >=, otherwise <.
The operand requires that the value being matched implements FileTypeIfc and
FileSizeIfc.

### Func FileSizeLarger
```go
func FileSizeLarger(n, v string) boolexpr.Operand
```

### Func FileSizeSmaller
```go
func FileSizeSmaller(n, v string) boolexpr.Operand
```

### Func FileType
```go
func FileType(opname string, typ string) boolexpr.Operand
```
FileType returns a 'file type' operand. It is not validated until a
matcher.T is created using New. Supported file types are (as per the unix
find command):
  - f for regular files
  - d for directories
  - l for symbolic links
  - x executable regular files

It requires that the value being matched implements FileTypeIfc for types d,
f and l and FileModeIfc for type x.

### Func Glob
```go
func Glob(opname string, pat string, caseInsensitive bool) boolexpr.Operand
```
Glob provides a glob operand (optionally case insensitive, in which case the
value it is being against will be converted to lower case before the match
is evaluated). The pattern is not validated until a matcher.T is created.
It requires that the value being matched implements NameIfc and/or PathIfc.
The NameIfc interface is used first, if the value does not implement NameIfc
or the glob evaluates to false, then PathIfc is used.

### Func New
```go
func New() *boolexpr.Parser
```
New returns a boolexpr.Parser with the following operands registered:
  - "name": case sensitive Glob
  - "iname", case insensitive Glob
  - "re", Regxp
  - "type", FileType
  - "newer", NewerThan
  - "dir-larger", DirSizeGreater
  - "dir-smaller", DirSizeSmaller
  - "file-larger", FileSizeGreater
  - "file-smaller", FileSizeSmaller

### Func NewDirSizeLarger
```go
func NewDirSizeLarger(n, v string) boolexpr.Operand
```
NewDirSizeLarger returns a boolexpr.Operand that returns true if the
expression value implements DirSizeIfc and the number of entries in the
directory is greater than the specified value.

### Func NewDirSizeSmaller
```go
func NewDirSizeSmaller(n, v string) boolexpr.Operand
```
NewDirSizeSmaller is like NewDirSizeLarger but returns true if the number of
entries is smaller or equal than the specified value.

### Func NewFileSizeLarger
```go
func NewFileSizeLarger(n, v string) boolexpr.Operand
```
NewFileSizeLarger returns a boolexpr.Operand that returns true if the
expression value implements DirSizeIfc and the number of entries in the
directory is greater than the specified value.

### Func NewFileSizeSmaller
```go
func NewFileSizeSmaller(n, v string) boolexpr.Operand
```
NewFileSizeSmaller is like NewFileSizeLarger but returns true if the number
of entries is smaller or equal than the specified value.

### Func NewFileType
```go
func NewFileType(n, v string) boolexpr.Operand
```
NewFileType returns a boolexpr.Operand that matches a file type.
The expression value must implement FileTypeIfc for types d, f and l and
FileModeIfc for type x.

### Func NewGlob
```go
func NewGlob(n, v string) boolexpr.Operand
```
NewGlob returns a case sensitive boolexpr.Operand that matches a glob
pattern. The expression value must implement NameIfc.

### Func NewGroup
```go
func NewGroup(name, value string, parser XAttrParser) boolexpr.Operand
```
NewGroup returns an operand that compares the group id of the value being
evaluated with the supplied group id or name. The supplied IDLookup is used
to convert the supplied text into a group id. The value being evaluated must
implement the XAttrIfc interface.

### Func NewIGlob
```go
func NewIGlob(n, v string) boolexpr.Operand
```
NewIGlob is a case-insensitive version of NewGlob. The expression value must
implement NameIfc.

### Func NewNewerThan
```go
func NewNewerThan(n, v string) boolexpr.Operand
```
NewNewerThan returns a boolexpr.Operand that matches a time that is
newer than the specified time. The time is specified in time.RFC3339,
time.DateTime, time.TimeOnly or time.DateOnly formats. The expression value
must implement ModTimeIfc.

### Func NewRegexp
```go
func NewRegexp(n, v string) boolexpr.Operand
```
NewRegexp returns a boolexpr.Operand that matches a regular expression.
The expression value must implement NameIfc.

### Func NewUser
```go
func NewUser(name, value string, parser XAttrParser) boolexpr.Operand
```
NewUser returns an operand that compares the user id of the value being
evaluated with the supplied user id or name. The supplied IDLookup is used
to convert the supplied text into a user id. The value being evaluated must
implement the XAttrIfc interface.

### Func NewerThanParsed
```go
func NewerThanParsed(opname string, value string) boolexpr.Operand
```
NewerThanParsed returns a 'newer than' operand. It is not validated until
a matcher.T is created using New. The time must be expressed as one of
time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly. Due to the nature
of the parsed formats fine grained time comparisons are not possible.

It requires that the value being matched implements ModTimeIfc.

### Func NewerThanTime
```go
func NewerThanTime(opname string, when time.Time) boolexpr.Operand
```
NewerThanTime returns a 'newer than' operand with the specified time.
This should be used in place of NewerThanFormat when fine grained time
comparisons are required.

It requires that the value bein matched implements Mod

### Func ParseGroupnameOrID
```go
func ParseGroupnameOrID(nameOrID string, lookup func(name string) (user.Group, error)) (file.XAttr, error)
```
ParseGroupnameOrID returns a file.XAttr that represents the supplied name or
ID.

### Func ParseUsernameOrID
```go
func ParseUsernameOrID(nameOrID string, lookup func(name string) (userid.IDInfo, error)) (file.XAttr, error)
```
ParseUsernameOrID returns a file.XAttr that represents the supplied name or
ID.

### Func Regexp
```go
func Regexp(opname string, re string) boolexpr.Operand
```
Regexp returns a regular expression operand. It is not compiled until a
matcher.T is created using New. It requires that the value being matched
implements PathIfc.

### Func XAttr
```go
func XAttr(opname, value, doc string,
	prepare XAttrParser,
	eval func(opVal, val file.XAttr) bool) boolexpr.Operand
```
XAttr returns an operand that compares an xattr value with the xattr value
of the value being evaluated.



## Types
### Type DirSizeIfc
```go
type DirSizeIfc interface {
	NumEntries() int64
}
```
DirSizeIfc must be implemented by any values that are used with the DirSize
operand.


### Type FileModeIfc
```go
type FileModeIfc interface {
	Mode() fs.FileMode
}
```
FileModeIfc must be implemented by any values that are used with the
Filetype operand for type x.


### Type FileSizeIfc
```go
type FileSizeIfc interface {
	Size() int64
}
```
FileSizeIfc must be implemented by any values that are used with the
FileSize operand.


### Type FileTypeIfc
```go
type FileTypeIfc interface {
	Type() fs.FileMode
}
```
FileTypeIfc must be implemented by any values that are used with the
Filetype operand for types f, d or l.


### Type ModTimeIfc
```go
type ModTimeIfc interface {
	ModTime() time.Time
}
```
ModTimeIfc must be implemented by any values that are used with the
NewerThan operand.


### Type NameIfc
```go
type NameIfc interface {
	Name() string
}
```
NameIfc and/or PathIfc must be implemented by any values that are used with
the Glob operands.


### Type PathIfc
```go
type PathIfc interface {
	Path() string
}
```
PathIfc must be implemented by any values that are used with the Regexp
operand optionally for the Glob operand.


### Type XAttrIfc
```go
type XAttrIfc interface {
	XAttr() file.XAttr
}
```
XAttrIfc must be implemented by any values that are used with the XAttr
operand.


### Type XAttrParser
```go
type XAttrParser func(text string) (file.XAttr, error)
```





