# Package [cloudeng.io/file/largefile](https://pkg.go.dev/cloudeng.io/file/largefile?tab=doc)

```go
import cloudeng.io/file/largefile
```


## Functions
### Func NewBackoff
```go
func NewBackoff(initial time.Duration, steps int) ratecontrol.Backoff
```
NewBackoff creates a new backoff instance that implements an exponential
backoff algorithm unless the RetryResponse specifies a specific backoff
duration. The backoff will continue for the specified number of steps,
after which it will return true to indicate that no more retries should be
attempted.

### Func ReserveSpace
```go
func ReserveSpace(ctx context.Context, filename string, size int64, blockSize, concurrency int) error
```
ReserveSpace creates a file with the specified filename and allocates the
specified size bytes to it. It verifies that the file was created with the
requested storage allocated. On systems that support space reservation,
such as Linux, space is reserved accordingly, on others data is written to
the file to ensure that the space is allocated. The intent is to ensure that
a download operations never fails because of insufficient local space once
it has been initiated.

### Func TestLocalDownloadCache_Get_Errors
```go
func TestLocalDownloadCache_Get_Errors(t *testing.T)
```

### Func TestLocalDownloadCache_Put_Errors
```go
func TestLocalDownloadCache_Put_Errors(t *testing.T)
```



## Types
### Type ByteRange
```go
type ByteRange struct {
	From int64 // Inclusive start of the range.
	To   int64 // Exclusive end of the range.
}
```
ByteRange represents a range of bytes in a file. The range is inclusive
of the 'From' byte and the 'To' byte as per the HTTP Range header
specification.

### Methods

```go
func (br ByteRange) Size() int64
```


```go
func (br ByteRange) String() string
```




### Type ByteRanges
```go
type ByteRanges struct {
	// contains filtered or unexported fields
}
```
ByteRange represents a collection of equally sized, contiguous, byte ranges
that can be used to track which parts of a file to download or that have
been downloaded.

### Functions

```go
func NewByteRanges(contentSize int64, blockSize int) *ByteRanges
```
NewByteRanges creates a new ByteRanges instance with the specified content
size and block size. The content size is the total size of the file in
bytes, and the block size is the size of each byte range in bytes.



### Methods

```go
func (br *ByteRanges) BlockSize() int
```


```go
func (br *ByteRanges) ContentLength() int64
```


```go
func (br *ByteRanges) IsClear(pos int64) bool
```
IsClear checks if the byte range for the specified position is clear.


```go
func (br *ByteRanges) IsSet(pos int64) bool
```
IsSet checks if the byte range for the specified position is set.


```go
func (br *ByteRanges) MarshalJSON() ([]byte, error)
```
MarshalJSON implements the json.Marshaler interface for ByteRanges.


```go
func (br *ByteRanges) NextClear(start int) iter.Seq[ByteRange]
```
NextClear returns an iterator for the next clear byte range starting from
'start'.


```go
func (br *ByteRanges) NextSet(start int) iter.Seq[ByteRange]
```
NextSet returns an iterator for the next set byte range starting from
'start'.


```go
func (br *ByteRanges) Set(pos int64)
```
Set marks the byte range for the specified position as set. It has no effect
if the position is out of bounds.


```go
func (br *ByteRanges) UnmarshalJSON(data []byte) error
```
UnmarshalJSON implements the json.Unmarshaler interface for ByteRanges.




### Type CachingDownloader
```go
type CachingDownloader struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewCachingDownloader(file Reader, cache DownloadCache, opts ...DownloadOption) *CachingDownloader
```



### Methods

```go
func (dl *CachingDownloader) Run(ctx context.Context) error
```
Run initializes the download.




### Type ChecksumType
```go
type ChecksumType int
```
ChecksumType represents the type of checksum used for file integrity
verification.

### Constants
### NoChecksum, MD5, SHA1, CRC32C
```go
NoChecksum ChecksumType = iota
MD5
SHA1
CRC32C

```




### Type DownloadCache
```go
type DownloadCache interface {
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)
	Outstanding() iter.Seq[ByteRange]
	Cached() iter.Seq[ByteRange]
	Put(r ByteRange, data []byte) error
	Get(r ByteRange, data []byte) error
}
```
DownloadCache is an interface for caching byte ranges of large files to
support resumable downloads.


### Type DownloadOption
```go
type DownloadOption func(*downloadOptions)
```

### Functions

```go
func WithDownloadConcurrency(n int) DownloadOption
```


```go
func WithDownloadLogger(logger *slog.Logger) DownloadOption
```


```go
func WithDownloadProgress(progress chan<- Progress) DownloadOption
```


```go
func WithDownloadRateController(rc *ratecontrol.Controller) DownloadOption
```


```go
func WithVerifyChecksum(verify bool) DownloadOption
```




### Type LocalDownloadCache
```go
type LocalDownloadCache struct {
	// contains filtered or unexported fields
}
```
LocalDownloadCache is a concrete implementation of RangeCache that uses a
local file to cache byte ranges of large files. It allows for concurrent
access.

### Functions

```go
func NewLocalDownloadCache(filename, indexFileName string, contentSize int64, blockSize int) (*LocalDownloadCache, error)
```



### Methods

```go
func (c *LocalDownloadCache) Cached() iter.Seq[ByteRange]
```


```go
func (c *LocalDownloadCache) ContentLengthAndBlockSize() (int64, int)
```


```go
func (c *LocalDownloadCache) Get(r ByteRange, data []byte) ([]byte, error)
```


```go
func (c *LocalDownloadCache) Outstanding() iter.Seq[ByteRange]
```


```go
func (c *LocalDownloadCache) Put(r ByteRange, data []byte) error
```




### Type Progress
```go
type Progress struct {
	BytesDownloaded  int64 // Total bytes downloaded so far.
	BlocksDownloaded int64 // Total blocks downloaded so far.
	TotalSize        int64 // Total size of the file in bytes.
	TotalBlocks      int64 // Total number of blocks to download.
}
```


### Type Reader
```go
type Reader interface {
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize(ctx context.Context) (int64, int, error)

	// Checksum returns the checksum type and the checksum value for the file,
	// if none are available then it returns NoChecksum and an empty string.
	Checksum(ctx context.Context) (ChecksumType, string, error)

	// GetReader retrieves a byte range from the file and returns
	// a reader that can be used to access that data range. In addition to the
	// error, the RetryResponse is returned which indicates whether the
	// operation can be retried and the duration to wait before retrying.
	GetReader(ctx context.Context, from, to int64) (io.ReadCloser, RetryResponse, error)
}
```
Reader provides support for downloading very large files efficiently
concurrently and to allow for resumption of partial downloads.


### Type RetryResponse
```go
type RetryResponse interface {
	// IsRetryable checks if the error is retryable.
	IsRetryable() bool

	// BackoffDuration returns true if a specific backoff duration is specified
	// in the response, in which case the duration is returned. If false
	// no specific backoff duration is requested and the backoff algorithm
	// should fallback to something appropriate, such as exponential backoff.
	BackoffDuration() (bool, time.Duration)
}
```
RetryResponse allows the caller to determine whether an operation that
failed with a retryable error can be retried and how long to wait before
retrying the operation.


### Type Uploader
```go
type Uploader struct{}
```





