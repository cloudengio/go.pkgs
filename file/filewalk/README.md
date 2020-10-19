# Package [cloudeng.io/file/filewalk](https://pkg.go.dev/cloudeng.io/file/filewalk?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/filewalk)](https://goreportcard.com/report/cloudeng.io/file/filewalk)

```go
import cloudeng.io/file/filewalk
```

Package filewalk provides support for concurrent traversal of file system
directories and files. It can traverse any filesytem that implements the
Filesystem interface and is intended to be usable with cloud storage systems
as AWS S3 or GCP's Cloud Storage. All compatible systems must implement some
sort of hierarchical naming scheme, whether it be directory based (as per
Unix/POSIX filesystems) or by convention (as per S3).

## Types
### Type Contents
```go
type Contents struct {
	Path     string `json:"p,omitempty"` // The name of the level being scanned.
	Children []Info `json:"c,omitempty"` // Info on each of the next levels in the hierarchy.
	Files    []Info `json:"f,omitempty"` // Info for the files at this level.
	Err      error  `json:"e,omitempty"` // Non-nil if an error occurred.
}
```
Contents represents the contents of the filesystem at the level represented
by Path.


### Type ContentsFunc
```go
type ContentsFunc func(ctx context.Context, prefix string, info *Info, ch <-chan Contents) ([]Info, error)
```
ContentsFunc is the type of the function that is called to consume the
results of scanning a single level in the filesystem hierarchy. It should
read the contents of the supplied channel until that channel is closed.
Errors, such as failing to access the prefix, are delivered over the
channel.


### Type Database
```go
type Database interface {
	// Set stores the specified information in the database taking care to
	// update all metrics. If PrefixInfo specifies a UserID then the metrics
	// associated with that user will be updated in addition to global ones.
	// Metrics are updated approriately for
	Set(ctx context.Context, prefix string, info *PrefixInfo) error
	// Get returns the information stored for the specified prefix. It will
	// return false if the entry does not exist in the database but with
	// a nil error.
	Get(ctx context.Context, prefix string, info *PrefixInfo) (bool, error)
	// Save saves the database to persistent storage.
	Save(ctx context.Context) error
	// Close will first Save and then release resources associated with the database.
	Close(ctx context.Context) error
	// UserIDs returns the current set of userIDs known to the database.
	UserIDs(ctx context.Context) ([]string, error)
	// Metrics returns the names of the supported metrics.
	Metrics() []MetricName
	// Total returns the total (ie. sum) for the requested metric.
	Total(ctx context.Context, name MetricName, opts ...MetricOption) (int64, error)
	// TopN returns the top-n values for the requested metric.
	TopN(ctx context.Context, name MetricName, n int, opts ...MetricOption) ([]Metric, error)
	// NewScanner creates a scanner that will start at the specified prefix
	// and scan at most limit items; a limit of 0 will scan all available
	// items.
	NewScanner(prefix string, limit int, opts ...ScannerOption) DatabaseScanner
}
```
Database is the interface to be implemented by a database suitable for use
with filewalk.


### Type DatabaseOption
```go
type DatabaseOption func(o *DatabaseOptions)
```
DatabaseOption represent a specific option common to all databases.

### Functions

```go
func ReadOnly() DatabaseOption
```
ReadOnly requests that the database be opened in read only mode.


```go
func ResetStats() DatabaseOption
```
ResetStats requests that the database reset its statistics when opened.




### Type DatabaseOptions
```go
type DatabaseOptions struct {
	ResetStats bool
	ReadOnly   bool
}
```
DatabaseOptions represents options common to all database implementations.


### Type DatabaseScanner
```go
type DatabaseScanner interface {
	Scan(ctx context.Context) bool
	PrefixInfo() (string, *PrefixInfo)
	Err() error
}
```
DatabaseScanner implements an idiomatic go scanner.


### Type Error
```go
type Error struct {
	Path string
	Op   string
	Err  error
}
```
Error implements error and provides additional detail on the error
encountered.

### Methods

```go
func (e *Error) Error() string
```
Error implements error.




### Type FileMode
```go
type FileMode uint32
```
FileMode represents meta data about a single file, including its
permissions. Not all underlying filesystems may support the full set of
UNIX-style permissions.

### Constants
### ModePrefix, ModeLink, ModePerm
```go
ModePrefix FileMode = FileMode(os.ModeDir)
ModeLink FileMode = FileMode(os.ModeSymlink)
ModePerm FileMode = FileMode(os.ModePerm)

```



### Methods

```go
func (fm FileMode) String() string
```
String implements stringer.




### Type Filesystem
```go
type Filesystem interface {
	// Stat obtains Info for the specified path.
	Stat(ctx context.Context, path string) (Info, error)

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// List will send all of the contents of path over the supplied channel.
	List(ctx context.Context, path string, ch chan<- Contents)

	// IsPermissionError returns true if the specified error, as returned
	// by the filesystem's implementation, is a result of a permissions error.
	IsPermissionError(err error) bool

	// IsNotExist returns true if the specified error, as returned by the
	// filesystem's implementation, is a result of the object not existing.
	IsNotExist(err error) bool
}
```
Filesystem represents the interface that is implemeted for filesystems to be
traversed/scanned.

### Functions

```go
func LocalFilesystem(scanSize int) Filesystem
```




### Type Info
```go
type Info struct {
	Name    string    // base name of the file
	UserID  string    // user id as returned by the underlying system
	Size    int64     // length in bytes
	ModTime time.Time // modification time
	Mode    FileMode  // permissions, directory or link.
	// contains filtered or unexported fields
}
```
Info represents the information that can be retrieved for a single file or
prefix.

### Methods

```go
func (i Info) IsLink() bool
```
IsLink returns true for a symbolic or other form of link.


```go
func (i Info) IsPrefix() bool
```
IsPrefix returns true for a prefix.


```go
func (i Info) Perms() FileMode
```
Perms returns UNIX-style permissions.


```go
func (i Info) Sys() interface{}
```
Sys returns the underlying, if available, data source.




### Type Metric
```go
type Metric struct {
	Prefix string
	Value  int64
}
```
Metric represents a value associated with a prefix.


### Type MetricName
```go
type MetricName string
```
MetricName names a particular metric supported by instances of Database.

### Constants
### TotalFileCount, TotalPrefixCount, TotalDiskUsage
```go
// TotalFileCount refers to the total # of files in the database.
TotalFileCount MetricName = "totalFileCount"
// TotalPrefixCount refers to the total # of prefixes/directories in
// the database. For cloud based filesystems the prefixes are likely
// purely naming conventions as opposed to local filesystem directories.
TotalPrefixCount MetricName = "totalPrefixCount"
// TotalDiskUsage refers to the total disk usage of the files and prefixes
// in the database taking the filesystems block size into account.
TotalDiskUsage MetricName = "totalDiskUsage"

```




### Type MetricOption
```go
type MetricOption func(o *MetricOptions)
```
MetricOption is used to request particular metrics, either per-user or
global to the entire database.

### Functions

```go
func Global() MetricOption
```


```go
func UserID(userID string) MetricOption
```




### Type MetricOptions
```go
type MetricOptions struct {
	Global bool
	UserID string
}
```
MetricOptions is configured by instances of MetricOption.


### Type Option
```go
type Option func(o *options)
```
Option represents options accepted by Walker.

### Functions

```go
func ChanSize(n int) Option
```
ChanSize can be used to set the size of the channel used to send results to
ResultsFunc. It defaults to being unbuffered.


```go
func Concurrency(n int) Option
```
Concurreny can be used to change the degree of concurrency used. The default
is to use all available CPUs.




### Type PrefixFunc
```go
type PrefixFunc func(ctx context.Context, prefix string, info *Info, err error) (stop bool, children []Info, returnErr error)
```
PrefixFunc is the type of the function that is called to determine if a
given level in the filesystem hiearchy should be further examined or
traversed. If stop is true then traversal stops at this point, however if a
list of children is returned, they will be traversed directly rather than
obtaining the children from the filesystem. This allows for both exclusions
and incremental processing in conjunction with a database t be implemented.


### Type PrefixInfo
```go
type PrefixInfo struct {
	ModTime   time.Time
	Size      int64
	UserID    string
	Mode      FileMode
	Children  []Info
	Files     []Info
	DiskUsage int64 // DiskUsage is the total amount of storage required for the files under this prefix taking the filesystem's layout/block size into account.
	Err       string
}
```
PrefixInfo represents information on a given prefix.

### Methods

```go
func (pi *PrefixInfo) GobDecode(buf []byte) error
```
GobDecode implements gob.Decoder.


```go
func (pi PrefixInfo) GobEncode() ([]byte, error)
```
GobEncode implements gob.Encoder.




### Type ScannerOption
```go
type ScannerOption func(so *ScannerOptions)
```
ScannerOption represent a specific option common to all scanners.

### Functions

```go
func KeysOnly() ScannerOption
```


```go
func RangeScan() ScannerOption
```


```go
func ScanDescending() ScannerOption
```
ScanDescending requests a descending scan, the default is ascending.


```go
func ScanErrors() ScannerOption
```


```go
func ScanLimit(l int) ScannerOption
```




### Type ScannerOptions
```go
type ScannerOptions struct {
	Descending bool
	RangeScan  bool
	KeysOnly   bool
	ScanErrors bool
	ScanLimit  int
}
```
ScannerOptions represents the options common to all scanner implementations.


### Type Walker
```go
type Walker struct {
	// contains filtered or unexported fields
}
```
Walker implements the filesyste walk.

### Functions

```go
func New(filesystem Filesystem, opts ...Option) *Walker
```
New creates a new Walker instance.



### Methods

```go
func (w *Walker) Walk(ctx context.Context, prefixFn PrefixFunc, contentsFn ContentsFunc, roots ...string) error
```
Walk traverses the hierarchies specified by each of the roots calling
prefixFn and contentsFn as it goes. prefixFn will always be called before
contentsFn for the same prefix, but no other ordering guarantees are
provided.







