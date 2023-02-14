# Package [cloudeng.io/file/filetestutil](https://pkg.go.dev/cloudeng.io/file/filetestutil?tab=doc)

```go
import cloudeng.io/file/filetestutil
```


## Functions
### Func CompareFS
```go
func CompareFS(a, b file.FS) error
```
CompareFS returns nil if the two instances of fs.FS contain exactly the same
files and file contents.

### Func Contents
```go
func Contents(fs file.FS) map[string][]byte
```
Contents returns the contents stored in the mock fs.FS.

### Func NewFile
```go
func NewFile(rd io.ReadCloser, info fs.FileInfo) fs.File
```

### Func NewMockFS
```go
func NewMockFS(opts ...FSOption) file.FS
```
NewMockFS returns an new mock instance of file.FS as per the specified
options.



## Types
### Type BufferCloser
```go
type BufferCloser struct {
	*bytes.Buffer
}
```
BufferCloser adds an io.Closer to bytes.Buffer.

### Methods

```go
func (bc *BufferCloser) Close() error
```




### Type FSOption
```go
type FSOption func(o *fsOptions)
```
FSOption represents an option to configure a new mock instance of fs.FS.

### Functions

```go
func FSErrorOnly(err error) FSOption
```
FSErrorOnly requests a mock FS that always returns err.


```go
func FSScheme(s string) FSOption
```


```go
func FSWithConstantContents(val []byte, repeat int) FSOption
```
FSWithConstantContents requests a mock FS that will return files of a random
size (up to maxSize) with random contents.


```go
func FSWithRandomContents(src rand.Source, maxSize int) FSOption
```
FSWithRandomContents requests a mock FS that will return files of a random
size (up to maxSize) with random contents.


```go
func FSWithRandomContentsAfterRetry(src rand.Source, maxSize, numRetries int, err error) FSOption
```
FSWithRandomContentsAfterRetry is like FSWithRandomContents but will return
err, numRetries times before succeeding.




### Type WriteFS
```go
type WriteFS struct {
	sync.Mutex
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewWriteFS() *WriteFS
```



### Methods

```go
func (wfs *WriteFS) Create(ctx context.Context, name string, filemode fs.FileMode) (io.WriteCloser, error)
```


```go
func (wfs *WriteFS) Open(name string) (fs.File, error)
```


```go
func (wfs *WriteFS) OpenCtx(ctx context.Context, name string) (fs.File, error)
```


```go
func (mfs *WriteFS) Scheme() string
```







