# Package [cloudeng.io/file/content/stores](https://pkg.go.dev/cloudeng.io/file/content/stores?tab=doc)

```go
import cloudeng.io/file/content/stores
```


## Types
### Type Async
```go
type Async struct {
	// contains filtered or unexported fields
}
```
AsyncWrite represents a store for objects with asynchronous writes. Reads
are synchronous. The caller must ensure that Finish is called to ensure that
all writes have completed.

### Functions

```go
func NewAsync(fs content.FS, concurrency int) *Async
```
NewAsync returns a new instance of Async with the specified concurrency.
If concurrency is less than or equal to zero, the number of CPUs is used.



### Methods

```go
func (s *Async) EraseExisting(ctx context.Context, root string) error
```
EraseExisting deletes all contents of the store beneath root.


```go
func (s *Async) FS() content.FS
```


```go
func (s *Async) Finish(context.Context) error
```
Finish waits for all queued writes to complete and returns any errors
encountered during the writes.


```go
func (s *Async) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error)
```
Read retrieves the object type and serialized data at the specified prefix
and name from the store. The caller is responsible for using the returned
type to decode the data into an appropriate object.


```go
func (s *Async) ReadV(ctx context.Context, prefix string, names []string, fn ReadFunc) error
```
ReadV retrieves the objects with the specified names from the store and
calls fn for each object. The read operations are performed concurrently.


```go
func (s *Async) Write(ctx context.Context, prefix, name string, data []byte) error
```
Write queues a write request for the specified prefix and name in the store.
There is no guarantee that the write will have completed when this method
returns. The error code returned is an indication that the write request was
queued and will only ever context.Canceled if non-nil.




### Type ReadFunc
```go
type ReadFunc func(ctx context.Context, prefix, name string, typ content.Type, data []byte, err error) error
```
ReadFunc is called by ReadV for each object read from the store. If the read
operation returned an error it is passed to ReadFunc and if then returned by
ReadFunc it will cause the entire ReadV operation to terminate and return an
error.


### Type Sync
```go
type Sync struct {
	// contains filtered or unexported fields
}
```
Sync represents a synchronous store for objects, ie. it implements
content.ObjectStore. It uses an instance of content.FS to store and retrieve
objects.

### Functions

```go
func NewSync(fs content.FS) *Sync
```
NewSync returns a new instance of Sync backed by the supplied content.FS and
storing the specified objects encoded using the specified encodings.



### Methods

```go
func (s *Sync) EraseExisting(ctx context.Context, root string) error
```
EraseExisting deletes all contents of the store beneath root.


```go
func (s *Sync) FS() content.FS
```


```go
func (s *Sync) Finish(context.Context) error
```


```go
func (s *Sync) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error)
```
Read retrieves the object type and serialized data at the specified prefix
and name from the store. The caller is responsible for using the returned
type to decode the data into an appropriate object.


```go
func (s *Sync) ReadV(ctx context.Context, prefix string, names []string, fn ReadFunc) error
```
ReadV calls ReadFunc for each object read from the store synchronously.


```go
func (s *Sync) Write(ctx context.Context, prefix, name string, data []byte) error
```
Write stores the data at the specified prefix and name in the store.




### Type T
```go
type T interface {
	content.ObjectStore
	EraseExisting(ctx context.Context, root string) error
	FS() content.FS
	ReadV(ctx context.Context, prefix string, names []string, fn ReadFunc) error
	Finish(context.Context) error
}
```
T represents a common interface for both synchronous and asynchronous
stores.

### Functions

```go
func New(fs content.FS, concurrency int) T
```
New returns a new instance of T with the specified concurrency.







