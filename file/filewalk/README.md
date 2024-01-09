# Package [cloudeng.io/file/filewalk](https://pkg.go.dev/cloudeng.io/file/filewalk?tab=doc)

```go
import cloudeng.io/file/filewalk
```

Package filewalk provides support for concurrent traversal of file system
directories and files. It can traverse any filesytem that implements the
Filesystem interface and is intended to be usable with cloud storage systems
as AWS S3 or GCP's Cloud Storage. All compatible systems must implement some
sort of hierarchical naming scheme, whether it be directory based (as per
Unix/POSIX filesystems) or by convention (as per S3).

## Variables
### DefaultScanSize, DefaultConcurrentScans
```go
// DefaultScansize is the default ScanSize used when the WithScanSize
// option is not supplied.
DefaultScanSize = 1000
// DefaultConcurrentScans is the default number of prefixes/directories
// that will be scanned concurrently when the WithConcurrencyOption is
// is not supplied.
DefaultConcurrentScans = 100

```



## Types
### Type Configuration
```go
type Configuration struct {
	ConcurrentScans int
	ScanSize        int
}
```


### Type Entry
```go
type Entry struct {
	Name string
	Type fs.FileMode // Type is the Type portion of fs.FileMode
}
```

### Methods

```go
func (de Entry) IsDir() bool
```




### Type EntryList
```go
type EntryList []Entry
```

### Functions

```go
func EntriesFromInfoList(infos file.InfoList) EntryList
```



### Methods

```go
func (el EntryList) AppendBinary(data []byte) ([]byte, error)
```
AppendBinary appends a binary encoded instance of Info to the supplied byte
slice.


```go
func (el *EntryList) DecodeBinary(data []byte) ([]byte, error)
```
DecodeBinary decodes the supplied data into an InfoList and returns the
remaining data.


```go
func (el EntryList) MarshalBinary() ([]byte, error)
```
MarshalBinary implements encoding.BinaryMarshaler.


```go
func (el *EntryList) UnmarshalBinary(data []byte) (err error)
```
UnmarshalBinary implements encoding.BinaryUnmarshaler.




### Type Error
```go
type Error struct {
	Path string
	Err  error
}
```
Error implements error and provides additional detail on the error
encountered.

### Methods

```go
func (e *Error) As(target interface{}) bool
```
As implements errors.As.


```go
func (e Error) Error() string
```
Error implements error.


```go
func (e Error) Is(target error) bool
```
Is implements errors.Is.


```go
func (e Error) Unwrap() error
```
Unwrap implements errors.Unwrap.




### Type FS
```go
type FS interface {
	file.FS

	LevelScanner(path string) LevelScanner
}
```
FS represents the interface that is implemeted for filesystems to be
traversed/scanned.


### Type Handler
```go
type Handler[T any] interface {

	// Prefix is called to determine if a given level in the filesystem hiearchy
	// should be further examined or traversed. The file.Info is obtained via a call
	// to Lstat and hence will refer to a symlink itself if the prefix is a symlink.
	// If stop is true then traversal stops at this point. If a list of Entry's
	// is returned then this list is traversed directly rather than obtaining
	// the children from the filesystem. This allows for both exclusions and
	// incremental processing in conjunction with a database to be implemented.
	// Any returned is recorded, but traversal will continue unless stop is set.
	Prefix(ctx context.Context, state *T, prefix string, info file.Info, err error) (stop bool, children file.InfoList, returnErr error)

	// Contents is called, multiple times, to process the contents of a single
	// level in the filesystem hierarchy. Each such call contains at most the
	// number of items allowed for by the WithScanSize option. Note that
	// errors encountered whilst scanning the filesystem result in calls to
	// Done with the error encountered.
	Contents(ctx context.Context, state *T, prefix string, contents []Entry) (file.InfoList, error)

	// Done is called once calls to Contents have been made or if Prefix returned
	// an error. Done will always be called if Prefix did not return true for stop.
	// Errors returned by Done are recorded and returned by the Walk method.
	// An error returned by Done does not terminate the walk.
	Done(ctx context.Context, state *T, prefix string, err error) error
}
```
Handler is implemented by clients of Walker to process the results of
walking a filesystem hierarchy. The type parameter is used to instantiate a
state variable that is passed to each of the methods.


### Type LevelScanner
```go
type LevelScanner interface {
	Scan(ctx context.Context, n int) bool
	Contents() []Entry
	Err() error
}
```


### Type Option
```go
type Option func(o *options)
```
Option represents options accepted by Walker.

### Functions

```go
func WithConcurrentScans(n int) Option
```
WithConcurrentScans can be used to change the number of prefixes/directories
that can be scanned concurrently. The default is DefaultConcurrentScans.


```go
func WithScanSize(n int) Option
```
WithScanSize sets the number of prefix/directory entries to be scanned in a
single operation. The default is DefaultScanSize.




### Type Stats
```go
type Stats struct {
	SynchronousScans int64
}
```


### Type Status
```go
type Status struct {
	// SynchronousOps is the number of Scans that were performed synchronously
	// as a fallback when all available goroutines are already occupied.
	SynchronousScans int64

	// SlowPrefix is a prefix that took longer than a certain duration
	// to scan. ScanDuration is the time spent scanning that prefix to
	// date. A SlowPrefix may be reported as slow before it has completed
	// scanning.
	SlowPrefix   string
	ScanDuration time.Duration
}
```
Status is used to communicate the status of in-process Walk operation.


### Type Walker
```go
type Walker[T any] struct {
	// contains filtered or unexported fields
}
```
Walker implements the filesyste walk.

### Functions

```go
func New[T any](fs FS, handler Handler[T], opts ...Option) *Walker[T]
```
New creates a new Walker instance.



### Methods

```go
func (w *Walker[T]) Configuration() Configuration
```


```go
func (w *Walker[T]) Stats() Stats
```


```go
func (w *Walker[T]) Walk(ctx context.Context, roots ...string) error
```
Walk traverses the hierarchies specified by each of the roots calling
prefixFn and entriesFn as it goes. prefixFn will always be called before
entriesFn for the same prefix, but no other ordering guarantees are
provided.







