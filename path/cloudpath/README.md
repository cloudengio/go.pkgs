# Package [cloudeng.io/path/cloudpath](https://pkg.go.dev/cloudeng.io/path/cloudpath?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/path/cloudpath)](https://goreportcard.com/report/cloudeng.io/path/cloudpath)

```go
import cloudeng.io/path/cloudpath
```

Package cloudpath provides utility routines for working with paths across
both local and distributed storage systems. The set of schemes supported can
be extended by providing additional implementations of the Matcher function.
A cloudpath encodes two types of information:

    1. the path name itself which can be used to access the data it names.
    2. metadata about the where that filename is hosted.

For example, s3://my-bucket/a/b, contains the path '/my-bucket/a/b' as well
the indication that this path is hosted on S3. Most cloud storage systems
either use URI formats natively or support their use. Both AWS S3 and Google
Cloud Storage support URLs: eg. storage.cloud.google.com/bucket/obj.

cloudpath provides operations for extracting both metadata and the path
component, and operations for working with the extracted path directly. A
common usage is to determine the 'scheme' (eg. s3, windows, unix etc) of a
filename and to then operate on it appropriately. cloudpath represents a
'path' as a slice of strings to simplify often performed operations such as
finding common prefixes, suffixes that are aware of the structure of the
path. For example it should be possible to easily determine that
s3://bucket/a/b is a prefix of
https://s3.us-west-2.amazonaws.com/bucket/a/b/c.

All of the metadata for a path is represented using the Match type.

For manipulation, the path is converted to a cloudpath.T.

## Constants
### AWSS3, GoogleCloudStorage, UnixFileSystem, WindowsFileSystem
```go
// AWSS3 is the scheme for Amazon Web Service's S3 object store.
AWSS3 = "s3"
// GoogleCloudStorage is the scheme for Google's Cloud Storage object store.
GoogleCloudStorage = "GoogleCloudStorage"
// UnixFileSystem is the scheme for unix like systems such as linux, macos etc.
UnixFileSystem = "unix"
// WindowsFileSystem is the scheme for msdos and windows filesystems.
WindowsFileSystem = "windows"

```



## Functions
### Func HasPrefix
```go
func HasPrefix(path, prefix []string) bool
```
HasPrefix returns true if path has the specified prefix.

### Func Host
```go
func Host(path string) string
```
Host calls DefaultMatchers.Host(path).

### Func IsLocal
```go
func IsLocal(path string) bool
```
IsLocal calls DefaultMatchers.IsLocal(path).

### Func Parameters
```go
func Parameters(path string) map[string][]string
```
Parameters calls DefaultMatchers.Parameters(path).

### Func Path
```go
func Path(path string) (string, rune)
```
Path calls DefaultMatchers.Path(path).

### Func Scheme
```go
func Scheme(path string) string
```
Scheme calls DefaultMatchers.Scheme(path).

### Func Volume
```go
func Volume(path string) string
```
Volume calls DefaultMatchers.Volume(path).



## Types
### Type Match
```go
type Match struct {
	// Scheme uniquely identifies the filesystem being used, eg. s3 or windows.
	Scheme string
	// Local is true for local filesystems.
	Local bool
	// Host will be 'localhost' for local filesystems, the host encoded
	// in a URL or otherwise empty if there is no notion of a host.
	Host string
	// Volume will be the bucket or file system share for systems that support
	// that concept, or an empty string otherwise.
	Volume string
	// Path is the filesystem path or filename to the data. It may be a prefix
	// on a cloud based system or a directory on a local one.
	Path string
	// Separator is the filesystem separator (e.g / or \ for windows).
	Separator rune
	// Parameters are any parameters encoded in a URL/URI based name.
	Parameters map[string][]string
}
```
Match is the result of a successful match.

### Functions

```go
func AWSS3Matcher(p string) *Match
```
AWSS3Matcher implements Matcher for AWS S3 object names. It returns AWSS3
for its scheme result.


```go
func GoogleCloudStorageMatcher(p string) *Match
```
GoogleCloudStorageMatcher implements Matcher for Google Cloud Storage object
names. It returns GoogleCloudStorage for its scheme result.


```go
func UnixMatcher(p string) *Match
```
UnixMatcher implements Matcher for unix filenames. It returns UnixFileSystem
for its scheme result.


```go
func WindowsMatcher(p string) *Match
```
WindowsMatcher implements Matcher for Windows filenames. It returns
WindowsFileSystem for its scheme result.




### Type Matcher
```go
type Matcher func(p string) *Match
```
Matcher is the prototype for functions that parse the supplied path to
determine if it matches a specific scheme and then breaks out the metadata
encoded in the path. Matchers for local filesystems should return
"localhost" for the host.


### Type MatcherSpec
```go
type MatcherSpec []Matcher
```
MatcherSpec represents a set of Matchers that will be applied in order. The
ordering is important, the most specific matchers need to be applied first.
For example a matcher for Windows should precede that for a Unix filesystem
since the latter can accept filenames in Windows format.

### Variables
### DefaultMatchers
```go
DefaultMatchers MatcherSpec = []Matcher{
	AWSS3Matcher,
	GoogleCloudStorageMatcher,
	WindowsMatcher,
	UnixMatcher,
}

```
DefaultMatchers represents the built in set of Matchers.



### Methods

```go
func (ms MatcherSpec) Host(path string) string
```
Host returns the host component of the path if there is one.


```go
func (ms *MatcherSpec) IsLocal(path string) bool
```
IsLocal returns true if the path is for a local filesystem.


```go
func (ms MatcherSpec) Match(p string) *Match
```
Match applies all of the matchers in turn to match the supplied path.


```go
func (ms *MatcherSpec) Parameters(path string) map[string][]string
```
Parameters returns the parameters in path, if any. If no parameters are
present an empty (rather than nil), map is returned.


```go
func (ms MatcherSpec) Path(path string) (string, rune)
```
Path returns the path component of path and the separator to use for it.


```go
func (ms MatcherSpec) Scheme(path string) string
```
Scheme returns the portion of the path that precedes a leading '//' or ""
otherwise.


```go
func (ms MatcherSpec) Volume(path string) string
```
Volume returns the filesystem specific volume, if any, encoded in the path.




### Type T
```go
type T []string
```
T represents a cloudpath. Instances of T are created from native storage
system paths and/or URLs and are designed to retain the following
information.

    1. the path was absolute vs relative.
    2. the path was a prefix or a filepath.
    3. a path of zero length is represented as a nil slice and not an empty slice.

Redundant information is discarded:

    1. multiple consecutive instances of separator are treated as a single separator.

The resulting format is as follows:

    1. a relative path, ie. one that does not start with a separator has an
       empty string as the first item in the slice
    2. a path that ends with a separator has an empty string as the final component
       of the path

For example:

    ""         => []                 // empty
    "/"        => ["", ""]           // absolute, prefix, IsRoot is true
    "/abc"     => ["", "abc"]        // absolute, filepath
    "abc"      => ["abc"]            // relative, filepath
    "/abc/"    => ["", "abc", ""]    // absolute, prefix, IsRoot is false
    "abc/"     => ["abc", ""]        // relative, prefix

T is defined as a type rather than using []string directly to avoid clients
of this package misinterpreting the above rules and incorrectly manipulating
the string slice.

### Functions

```go
func LongestCommonPrefix(paths []T) T
```
LongestCommonPrefix returns the longest prefix common to the specified
cloudpaths.


```go
func LongestCommonSuffix(paths []T) T
```
LongestCommonSuffix returns the longest suffix common to the specified
cloudpaths.


```go
func Split(path string, separator rune) T
```
Split slices path into an instance of T.


```go
func SplitPath(path string) T
```
SplitPath calls Split with the results of cloudpath.Split(path).


```go
func TrimPrefix(path, prefix []string) T
```
TrimPrefix removes the specified prefix from path. It returns nil if path
and suffix are identical.



### Methods

```go
func (path T) AsFilepath() T
```
AsFilepath returns path as a filepath if it is not already one provided that
is not a root or empty.


```go
func (path T) AsPrefix() T
```
AsPrefix returns path as a path prefix if it is not already one.


```go
func (path T) Base() string
```
Base returns the 'base', or 'filename' component of path, ie. the last one.
If the path is a prefix then an empty string is returned.


```go
func (path T) HasSuffix(suffix T) bool
```
HasSuffix returns true if path has the specified suffix.


```go
func (path T) IsAbsolute() bool
```
IsAbsolute returns true if the components were derived from an absolute
path.


```go
func (path T) IsFilepath() bool
```
IsFilepath returns true if the path was derived from a filepath.


```go
func (path T) IsRoot() bool
```
IsRoot returns true if the path was a derived from the 'root', ie. a single
separator such as /.


```go
func (path T) Join(separator rune) string
```
Join creates a string path from the supplied components. It follows the
rules specified for Join. It is the inverse of Split, that is, newPath ==
origPath for:

    newPath = Join(sep, Split(origPath,sep)...)


```go
func (path T) Pop() (T, string)
```
Pop returns a new cloudpath.T with the trailing component removed and
returned. Pop on a path for which IsRoot is true will return the root again.
IsFilepath will always be false for the returned cloudpath.T.


```go
func (path T) Prefix() T
```
Prefix returns the prefix component of a path.


```go
func (path T) Push(p string) T
```
Push returns a new cloudpath.T with the supplied component appended.
IsFilePath will always be true for the returned value unless p is an empty
string in which case Push is equivalent to path.AsFilePath().


```go
func (path T) String() string
```
String implements stringer. It calls path.Join with / as the separator.


```go
func (path T) TrimSuffix(suffix T) T
```
TrimSuffix removes the specified suffix from path. It returns nil if path
and suffix are identical.






## Examples
### [ExampleScheme](https://pkg.go.dev/cloudeng.io/path/cloudpath?tab=doc#example-Scheme)




