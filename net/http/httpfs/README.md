# Package [cloudeng.io/net/http/httpfs](https://pkg.go.dev/cloudeng.io/net/http/httpfs?tab=doc)

```go
import cloudeng.io/net/http/httpfs
```


## Variables
### ErrNoRangeSupport
```go
ErrNoRangeSupport = &errNoRangeSupport{}

```



## Functions
### Func NewLargeFile
```go
func NewLargeFile(ctx context.Context, name string, opts ...LargeFileOption) (largefile.Reader, error)
```
OpenLargeFile opens a large file for concurrent reading using
file.largefile.Reader.



## Types
### Type FS
```go
type FS struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func New(client *http.Client, options ...Option) *FS
```
New creates a new instance of file.FS backed by http/https.



### Methods

```go
func (fs *FS) Base(p string) string
```


```go
func (fs *FS) IsNotExist(err error) bool
```


```go
func (fs *FS) IsPermissionError(err error) bool
```


```go
func (fs *FS) Join(components ...string) string
```


```go
func (fs *FS) Lstat(_ context.Context, _ string) (file.Info, error)
```
Lstat issues a head request but will not follow redirects.


```go
func (fs *FS) Open(name string) (fs.File, error)
```
Open implements fs.FS.


```go
func (fs *FS) OpenCtx(ctx context.Context, name string) (fs.File, error)
```
OpenCtx implements file.FS.


```go
func (fs *FS) Readlink(_ context.Context, _ string) (string, error)
```
Readlink returns the contents of a redirect without following it.


```go
func (fs *FS) Scheme() string
```
Scheme implements fs.FS.


```go
func (fs *FS) Stat(_ context.Context, _ string) (file.Info, error)
```
Stat issues a head request and will follow redirects.


```go
func (fs *FS) SysXAttr(existing any, merge file.XAttr) any
```


```go
func (fs *FS) XAttr(_ context.Context, _ string, info file.Info) (file.XAttr, error)
```




### Type LargeFile
```go
type LargeFile struct {
	// contains filtered or unexported fields
}
```
LargeFile implements largefile.Reader for large files accessed via HTTP.
Such files must support range requests, and the "Accept-Ranges" header must
be set to "bytes". If the server does not support range requests, it returns
ErrNoRangeSupport. A HEAD request is made to the file to determine its
content length and digest (if available). The file must be capable of being
read concurrently in blocks of a specified range. Partial reads are treated
as errors.

### Methods

```go
func (f *LargeFile) ContentLengthAndBlockSize() (int64, int)
```


```go
func (f *LargeFile) Digest() string
```


```go
func (f *LargeFile) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error)
```


```go
func (f *LargeFile) Name() string
```
Name implements largefile.Reader.




### Type LargeFileOption
```go
type LargeFileOption func(o *largeFileOptions)
```

### Functions

```go
func WithDetaultRetryDelay(delay time.Duration) LargeFileOption
```
WithDefaultRetryDelay sets the default retry delay for HTTP requests.
This is used when the server responds with a 503 Service Unavailable status
but does not provide a Retry-After header or that header cannot be parsed.
The default value is 1 minute.


```go
func WithLargeFileBlockSize(blockSize int) LargeFileOption
```
WithLargeFileBlockSize sets the block size for reading large files.


```go
func WithLargeFileLogger(slog *slog.Logger) LargeFileOption
```
WithLargeFileLogger sets the logger. If not set, a discard logger is used.


```go
func WithLargeFileTransport(transport *http.Transport) LargeFileOption
```
WithLargeFileTransport sets the HTTP transport for making requests, if not
set a simple default is used.




### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithHTTPScheme() Option
```




### Type Response
```go
type Response struct {
	// When the response was received.
	When time.Time

	// Fields copied from the http.Response.
	Headers                http.Header
	Trailers               http.Header
	ContentLength          int64
	StatusCode             int
	ProtoMajor, ProtoMinir int
	TransferEncoding       []string
}
```
Response is a redacted version of http.Response that can be marshaled using
gob.





