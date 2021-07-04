# Package [cloudeng.io/file/filewalk/localdb](https://pkg.go.dev/cloudeng.io/file/filewalk/localdb?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/filewalk/localdb)](https://goreportcard.com/report/cloudeng.io/file/filewalk/localdb)

```go
import cloudeng.io/file/filewalk/localdb
```

Package localdb provides an implementation of filewalk.Database that uses a
local key/value store currently based on github.com/recoilme/pudge.

## Variables
### ErrReadonly
```go
ErrReadonly = errors.New("database is opened in readonly mode")

```
ErrReadonly is returned if an attempt is made to write to a database opened
in read-only mode.



## Functions
### Func Open
```go
func Open(ctx context.Context, dir string, ifcOpts []filewalk.DatabaseOption, opts ...DatabaseOption) (filewalk.Database, error)
```



## Types
### Type Database
```go
type Database struct {
	// contains filtered or unexported fields
}
```
Database represents an on-disk database that stores information and
statistics for filesystem directories/prefixes. The database supports
read-write and read-only modes of access.

### Methods

```go
func (db *Database) Close(ctx context.Context) error
```


```go
func (db *Database) CompactAndClose(ctx context.Context) error
```


```go
func (db *Database) Delete(ctx context.Context, separator string, prefixes []string, recurse bool) (int, error)
```


```go
func (db *Database) DeleteErrors(ctx context.Context, prefixes []string) (int, error)
```


```go
func (db *Database) Get(ctx context.Context, prefix string, info *filewalk.PrefixInfo) (bool, error)
```


```go
func (db *Database) GroupIDs(ctx context.Context) ([]string, error)
```


```go
func (db *Database) Metrics() []filewalk.MetricName
```


```go
func (db *Database) NewScanner(prefix string, limit int, opts ...filewalk.ScannerOption) filewalk.DatabaseScanner
```


```go
func (db *Database) Save(ctx context.Context) error
```


```go
func (db *Database) Set(ctx context.Context, prefix string, info *filewalk.PrefixInfo) error
```


```go
func (db *Database) Stats() ([]filewalk.DatabaseStats, error)
```


```go
func (db *Database) TopN(ctx context.Context, name filewalk.MetricName, n int, opts ...filewalk.MetricOption) ([]filewalk.Metric, error)
```


```go
func (db *Database) Total(ctx context.Context, name filewalk.MetricName, opts ...filewalk.MetricOption) (int64, error)
```


```go
func (db *Database) UserIDs(ctx context.Context) ([]string, error)
```




### Type DatabaseOption
```go
type DatabaseOption func(o *Database)
```
DatabaseOption represents a specific option accepted by Open.

### Functions

```go
func LockStatusDelay(d time.Duration) DatabaseOption
```
LockStatusDelay sets the delay between checking the status of acquiring a
lock on the database.


```go
func SyncInterval(interval time.Duration) DatabaseOption
```
SyncInterval set the interval at which the database is to be persisted to
disk.


```go
func TryLock() DatabaseOption
```
TryLock returns an error if the database cannot be locked within the delay
period.




### Type ScanOption
```go
type ScanOption func(ks *Scanner)
```
ScanOption represents an option used when creating a Scanner.


### Type Scanner
```go
type Scanner struct {
	// contains filtered or unexported fields
}
```
Scanner allows for the contents of an instance of Database to be enumerated.
The database is organized as a key value store that can be scanned by range
or by prefix.

### Functions

```go
func NewScanner(db *Database, prefix string, limit int, ifcOpts []filewalk.ScannerOption, opts ...ScanOption) *Scanner
```
NewScanner returns a new instance of Scanner.



### Methods

```go
func (sc *Scanner) Err() error
```
Err rimplements filewalk.DatabaseScanner.


```go
func (sc *Scanner) PrefixInfo() (string, *filewalk.PrefixInfo)
```
PrefixInfo implements filewalk.DatabaseScanner.


```go
func (sc *Scanner) Scan(ctx context.Context) bool
```
Scan implements filewalk.DatabaseScanner.







