# Package [cloudeng.io/file/filewalk/asyncstat](https://pkg.go.dev/cloudeng.io/file/filewalk/asyncstat?tab=doc)

```go
import cloudeng.io/file/filewalk/asyncstat
```


## Variables
### DefaultAsyncStats, DefaultAsyncThreshold
```go
// DefaultAsyncStats is the default maximum number of async stats to be issued
// when WithAsyncStats is not specified.
DefaultAsyncStats = 100
// DefaultAsyncThreshold is the default value for the number of directory
// entries that must be present before async stats are used when
// WithAsyncThreshold is not specified.
DefaultAsyncThreshold = 10

```



## Types
### Type Configuration
```go
type Configuration struct {
	AsyncStats     int
	AsyncThreshold int
}
```


### Type ErrorLogger
```go
type ErrorLogger func(ctx context.Context, filename string, err error)
```
ErrorLogger is the type of function called when a Stat or Lstat return an
error.


### Type LatencyTracker
```go
type LatencyTracker interface {
	Before() time.Time
	After(time.Time)
}
```
LatencyTracker is used to track the latency of Stat or Lstat operations.


### Type Option
```go
type Option func(*options)
```
Option is used to configure an asyncstat.T.

### Functions

```go
func WithAsyncStats(stats int) Option
```
WithAsyncStats sets the total number of asynchronous stats to be issued.
The default is DefaultAsyncStats.


```go
func WithAsyncThreshold(threshold int) Option
```
WithAsyncThreshold sets the threshold at which asynchronous stats are used,
any directory with less than number of entries will be processed
synchronously. The default is DefaultAsyncThreshold.


```go
func WithErrorLogger(fn ErrorLogger) Option
```
WithErrorLogger sets the function to be called when an error is returned by
Stat or Lstat.


```go
func WithLStat() Option
```
WithLStat requests that fs.LStat be used instead of fs.Stat. This is the
default.


```go
func WithLatencyTracker(lt LatencyTracker) Option
```
WithLatencyTracker sets the latency tracker to be used.


```go
func WithStat() Option
```
WithStat requests that fs.Stat be used instead of fs.LStat.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T provides support for issuing asynchronous stat or lstat calls.

### Functions

```go
func New(fs filewalk.FS, opts ...Option) *T
```
New returns an aysncstat.T that uses the supplied filewalk.FS.



### Methods

```go
func (is *T) Configuration() Configuration
```


```go
func (is *T) Process(ctx context.Context, prefix string, entries []filewalk.Entry) (children, all file.InfoList, err error)
```
Process processes the supplied entries, returning the list of children as
filewalk.Entry and the list of stat/lstat results as a file.InfoList.







