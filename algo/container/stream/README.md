# Package [cloudeng.io/algo/container/stream](https://pkg.go.dev/cloudeng.io/algo/container/stream?tab=doc)

```go
import cloudeng.io/algo/container/stream
```

Package stream provides generic building blocks for scanning and iterating
over streams of items, such as the results of a paginated API, the rows of
a query or the entries of a directory. A stream is implemented by providing
a Pop or PopN function, and the Scanner and Iter built on it then fetch
items in batches, optionally ahead of their consumption, so that the
implementation need only say how to obtain the next items.

## Constants
### DefaultBatchSize, DefaultPrefetch
```go
const (
	// DefaultBatchSize is the number of items requested from a PopN at a time.
	DefaultBatchSize = 64
	// DefaultPrefetch is the number of batches fetched ahead of their use.
	DefaultPrefetch = 2
)
```



## Functions
### Func Iter
```go
func Iter[T any](ctx context.Context, popn PopN[T], opts ...Option) iter.Seq2[T, error]
```
Iter returns an iterator over the items of the stream implemented by popn,
as used with range. If the stream ends with an error it is yielded,
along with the zero value, as the last element. The items are prefetched
as per NewScanner, and the prefetching is stopped when the iteration ends,
whether or not it was run to the end.



## Types
### Type Option
```go
type Option func(*options)
```
Option represents an option for NewScanner and Iter.

### Functions

```go
func WithBatchSize(n int) Option
```
WithBatchSize sets the number of items requested from a PopN at a time,
DefaultBatchSize if n is less than one.


```go
func WithPrefetch(n int) Option
```
WithPrefetch sets the number of batches that are fetched, by a goroutine,
ahead of the batch being consumed, including one that is being fetched,
so that fetching overlaps with consuming. Zero disables prefetching, so that
the PopN is only called, from the caller's goroutine, when more items are
needed. The default is DefaultPrefetch.




### Type Pop
```go
type Pop[T any] func(ctx context.Context) (item T, empty bool, err error)
```
Pop returns the next item of a stream. The item is only valid if empty is
false and err is nil; empty is true when the stream has no more items.


### Type PopN
```go
type PopN[T any] func(ctx context.Context, items []T) (n int, empty bool, err error)
```
PopN fills items, from the start, with as many as len(items) of the next
items of a stream, and returns the number that it stored in n. Empty is true
if the stream has no more items beyond those n, which may be zero, so that
an implementation that finds the end of the stream whilst filling items
need not be called again. A PopN must make progress: it must not return
zero items, with empty false and no error, since that is a stream that never
ends. If err is not nil the n items are still valid, but the stream is over.

### Functions

```go
func PopNFrom[T any](pop Pop[T]) PopN[T]
```
PopNFrom returns a PopN that is implemented using pop, for streams that can
only supply one item at a time.




### Type Scanner
```go
type Scanner[T any] struct {
	// contains filtered or unexported fields
}
```
Scanner iterates over the items of a stream, in the manner of bufio.Scanner:

    sc := stream.NewScanner(popn)
    defer sc.Close()
    for sc.Scan(ctx) {
        item := sc.Data()
    }
    if err := sc.Err(); err != nil {
        ...
    }

A Scanner is not safe for concurrent use.

### Functions

```go
func NewScanner[T any](popn PopN[T], opts ...Option) *Scanner[T]
```
NewScanner returns a Scanner for the stream implemented by popn.
The goroutine that the Scanner uses for prefetching is started by the first
call to Scan, and runs until the stream is exhausted or Close is called.
Close must be called if the Scanner is not used to the end.



### Methods

```go
func (s *Scanner[T]) Close() error
```
Close stops prefetching, waiting for the goroutine that does so to exit, and
releases the resources of the Scanner. It is safe to call more than once.
Scan returns false after Close.


```go
func (s *Scanner[T]) Data() T
```
Data returns the item found by the most recent call to Scan.


```go
func (s *Scanner[T]) Err() error
```
Err returns the error, if any, that ended the scan: that returned by the
PopN, or the error of the context if it was done. It is nil if the stream
was exhausted, or the Scanner closed.


```go
func (s *Scanner[T]) Scan(ctx context.Context) bool
```
Scan advances to the next item, which is then available via Data. It returns
false when there are no more items, either because the stream has been
exhausted or because of an error, which Err returns.

ctx is passed to each call to the PopN that this call to Scan initiates,
which, when prefetching, is a call for a batch that will be consumed by a
later Scan. It must therefore remain valid for as long as the Scanner is
used; if it is done the scan ends with its error.






## Examples
### [ExampleIter](https://pkg.go.dev/cloudeng.io/algo/container/stream?tab=doc#example-Iter)

### [ExampleScanner](https://pkg.go.dev/cloudeng.io/algo/container/stream?tab=doc#example-Scanner)




