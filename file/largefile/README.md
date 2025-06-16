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

### Func NewFilesForCache
```go
func NewFilesForCache(ctx context.Context, filename, indexFileName string, contentSize int64, blockSize, concurrency int) error
```
NewFilesForCache creates a new cache file and an index file for caching byte
ranges of large files. It reserves space for the cache file and initializes
the index file with the specified content size and block size. It returns
an error if the files cannot be created or if the space cannot be reserved.
The index file is used to track which byte ranges have been written to the
cache. The cache file is used to store the actual data. The contentSize is
the total size of the file in bytes, blockSize is the preferred block size
for downloading the file, and concurrency is the number of concurrent writes
used to reserve space for the cache file on systems that require writing to
the file to reserve space (e.g., non-Linux systems).

### Func NumBlocks
```go
func NumBlocks(contentSize int64, blockSize int) int
```
NumBlocks calculates the number of blocks required to cover the content size
given the specified block size. It returns the number of blocks needed. If
the content size is not a multiple of the block size, it adds an additional
block to cover the remaining bytes.

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

### Func TestLocalDownloadCacheGetErrors
```go
func TestLocalDownloadCacheGetErrors(t *testing.T)
```

### Func TestLocalDownloadCachePutErrors
```go
func TestLocalDownloadCachePutErrors(t *testing.T)
```



## Types
### Type ByteRange
```go
type ByteRange struct {
	From int64 // Inclusive start of the range.
	To   int64 // Inclusive end of the range.
}
```
ByteRange represents a range of bytes in a file. The range is inclusive
of the 'From' byte and the 'To' byte as per the HTTP Range header
specification/convention.

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
ByteRange represents a collection of equally sized (apart from the last
range), contiguous, byte ranges that can be used to track which parts of a
file have or have not been 'processed', e.g downloaded, cached, uploaded
etc. The ranges are represented as a bitmap, where each bit corresponds to
a block of bytes of the specified size. The bitmap is used to efficiently
track which byte ranges are set (processed) and which are clear (not
processed).

### Functions

```go
func NewByteRanges(contentSize int64, blockSize int) *ByteRanges
```
NewByteRanges creates a new ByteRanges instance with the specified content
size and block size. The content size is the total size of the file in
bytes, and the block size is the size of each byte range in bytes.



### Methods

```go
func (br *ByteRanges) AllClear(start int) iter.Seq[ByteRange]
```
AllClear returns an iterator for all clear byte ranges starting from
'start'. A read lock is held while iterating over the byte ranges, hence
calling any other method, such as Set, which takes a write lock will block
until the iteration is complete. Use NextClear if finer-grained control is
needed.


```go
func (br *ByteRanges) AllSet(start int) iter.Seq[ByteRange]
```
AllSet returns an iterator for all set byte ranges starting from 'start'.
A read lock is held while iterating over the byte ranges, hence calling any
other method, such as Set, which takes a write lock will block until the
iteration is complete. Use NextSet if finer-grained control is needed.


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
func (br *ByteRanges) NextClear(start int, nbr *ByteRange) int
```
NextClear returns the next clear byte range starting from 'start'. It starts
searching from the specified start index and returns the index of the next
outstanding range which can be used to continue searching for the next
outstanding range. The index will be -1 if there are no more outstanding
ranges.

    for start := NextClear(0, &br); start >= 0; start = NextClear(start, &br) {
        // Do something with the byte range br.
    }


```go
func (br *ByteRanges) NextSet(start int, nbr *ByteRange) int
```
NextSet returns the next set byte range starting from 'start' and behaves
similarly to NextClear.


```go
func (br *ByteRanges) NumBlocks() int
```
NumBlocks returns the number of blocks required to cover the byte ranges
represented by this ByteRanges instance.


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
CachingDownloader is a downloader that caches streamed downloaded data to a
local cache and supports resuming downloads from where they left off.

### Functions

```go
func NewCachingDownloader(file Reader, cache DownloadCache, opts ...DownloadOption) *CachingDownloader
```
NewCachingDownloader creates a new CachingDownloader instance.



### Methods

```go
func (dl *CachingDownloader) Run(ctx context.Context) (DownloadStatus, error)
```
Run executes the downloaded process. If the downloader encounters any errors
it will return an




### Type DownloadCache
```go
type DownloadCache interface {
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)
	// NextOutstanding finds the next byte range that has not been cached
	// starting from the specified 'start' index. Its return value is either
	// -1 if there are no more outstanding ranges, or the value of the next
	// starting index to continue searching at.
	// To iterate over all outstanding ranges, call this method repeatedly
	// until it returns -1 as follows:
	//    for start := NextOutstanding(0, &br); start != -1; start = NextOutstanding(start, &br) {
	//        // Do something with the byte range br.
	//    }
	NextOutstanding(start int, br *ByteRange) int
	// NextCached finds the next byte range that has been cached in the same manner
	// as NextOutstanding.
	NextCached(start int, br *ByteRange) int
	// Complete returns true if all byte ranges have been cached.
	Complete() bool
	// Put writes the specified byte range to the cache.
	Put(r ByteRange, data []byte) error
	// Get reads the specified byte range from the cache into the provided data slice.
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
WithDownloadConcurrency sets the number of concurrent download goroutines.


```go
func WithDownloadLogger(logger *slog.Logger) DownloadOption
```
WithDownloadLogger sets the logger for the download.


```go
func WithDownloadProgress(progress chan<- DownloadState) DownloadOption
```
WithDownloadProgress sets the channel to report download progress.


```go
func WithDownloadRateController(rc ratecontrol.Limiter) DownloadOption
```
WithDownloadRateController sets the rate controller for the download.


```go
func WithDownloadWaitForCompletion(wait bool) DownloadOption
```
WithDownloadWaitForCompletion sets whether the download should iterate,
until the download is successfully completed, or return after one iteration.
An iteration represents a single pass through the download process whereby
every outstsanding byte range is attempted to be downloaded once with
retries. A download will either complete after any specified retries or be
left outstanding for the next iteration.




### Type DownloadState
```go
type DownloadState struct {
	CachedBytes      int64 // Total bytes cached.
	CachedBlocks     int64 // Total blocks cached.
	CacheErrors      int64 // Total number of errors encountered while caching.
	DownloadedBytes  int64 // Total bytes downloaded so far.
	DownloadedBlocks int64 // Total blocks downloaded so far.
	DownloadSize     int64 // Total size of the file in bytes.
	DownloadBlocks   int64 // Total number of blocks to download.
	DownloadRetries  int64 // Total number of retries made during the download.
	DownloadErrors   int64 // Total number of errors encountered during the download.
	Iterations       int64 // Number of iterations requiredd to complete the download.
}
```


### Type DownloadStatus
```go
type DownloadStatus struct {
	DownloadState
	Resumeable bool          // Indicates if the download can be re-run.
	Complete   bool          // Indicates if the download completed successfully.
	Duration   time.Duration // Total duration of the download.
}
```
DownloadStatus holds the status of a download operation, including the
progress made, whether the download is resumable, completed and the total
duration of operation.


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
func NewLocalDownloadCache(filename, indexFileName string) (*LocalDownloadCache, error)
```
NewLocalDownloadCache creates a new LocalDownloadCache instance. It opens
the cache file and loads the index file containing the byte ranges that have
been written to the cache. It returns an error if the files cannot be opened
or if the index file cannot be loaded. The cache file is used to store the
actual data, and the index file is used to track which byte ranges have been
written to the cache. The cache and index files must already exist and are
expected to be have been created using NewFilesForCache.



### Methods

```go
func (c *LocalDownloadCache) Close() error
```
Close implements DownloadCache.


```go
func (c *LocalDownloadCache) Complete() bool
```
Complete implements DownloadCache. It returns true if all byte ranges have
been cached, meaning there are no more uncached ranges.


```go
func (c *LocalDownloadCache) ContentLengthAndBlockSize() (int64, int)
```
ContentLengthAndBlockSize implements DownloadCache.


```go
func (c *LocalDownloadCache) Get(r ByteRange, data []byte) error
```
Get implements DownloadCache.


```go
func (c *LocalDownloadCache) NextCached(start int, br *ByteRange) int
```
NextCached implements DownloadCache. It returns the next, if any, cached
byte range starting from the specified index.


```go
func (c *LocalDownloadCache) NextOutstanding(start int, br *ByteRange) int
```
NextOutstanding implements DownloadCache. It returns the next, if any,
uncached byte range starting from the specified index.


```go
func (c *LocalDownloadCache) Put(r ByteRange, data []byte) error
```
Put implements DownloadCache.




### Type Reader
```go
type Reader interface {
	Name() string // Name returns the name of the file being read.

	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)

	// Digest returns the digest of the file, if available, the
	// format defined by RFC 9530's Repr-Digest header, eg.
	// Repr-Digest: sha-256=:d435Qo+nKZ+gLcUHn7GQtQ72hiBVAgqoLsZnZPiTGPk=:
	// An empty string is returned if no digest is available.
	Digest() string

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


### Type StreamingDownloadOption
```go
type StreamingDownloadOption func(*downloadStreamingOptions)
```

### Functions

```go
func WithVerifyDigest(verify bool) StreamingDownloadOption
```




### Type StreamingDownloader
```go
type StreamingDownloader struct {
	// contains filtered or unexported fields
}
```
StreamingDownloader is a downloader that streams data from a large file.
The downloader uses concurrent byte range requests to fetch data and then
serializes the responses into a single stream for reading.

### Functions

```go
func NewStreamingDownloader(file Reader, opts ...StreamingDownloadOption) *StreamingDownloader
```
NewStreamingDownloader creates a new StreamingDownloader instance.




### Type Uploader
```go
type Uploader struct{}
```





