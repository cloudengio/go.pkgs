# Package [cloudeng.io/file/download](https://pkg.go.dev/cloudeng.io/file/download?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/download)](https://goreportcard.com/report/cloudeng.io/file/download)

```go
import cloudeng.io/file/download
```

Package download provides a simple download mechanism that uses the fs.FS
container API to implement the actual download. This allows rate control,
retries and download management to be separated from the mechanism of the
actual download. Downloaders can be provided for http/https, AWS S3 or
any other local or cloud storage system for which an fs.FS implementation
exists.

## Types
### Type Downloaded
```go
type Downloaded struct {
	Request   Request
	Downloads []Result
}
```
Downloaded represents all of the downloads in response to a given request.


### Type Option
```go
type Option func(*options)
```
Option is used to configure the behaviour of a newly created Downloader.

### Functions

```go
func WithBackoffParameters(retryErr error, first time.Duration, steps int) Option
```
WithBackoffParameters enables an exponential backoff algorithm that
is triggered when the download fails in a way that is retryable. The
container (fs.FS) implementation must return an error that returns true for
errors.Is(err, retryErr). First defines the first backoff delay, which is
then doubled for every consecutive matching error until the download either
succeeds or the specified number of steps (attempted downloads) is exceeded
(the download is then deemed to have failed).


```go
func WithNumDownloaders(concurrency int) Option
```
WithNumDownloaders controls the number of concurrent downloads used.
If not specified the default of runtime.GOMAXPROCS(0) is used.


```go
func WithProgress(interval time.Duration, ch chan<- Progress, close bool) Option
```
WithProgress requests that progress messages are sent over the supplid
channel. If close is true the progress channel will be closed when the
downloader has finished. Close should be set to false if the same channel is
shared across multiplied downloader instances.


```go
func WithRequestsPerMinute(rpm int) Option
```
WithRequestsPerMinute sets the rate for download requests. If not specified
downloads will be initiated immediately.




### Type Progress
```go
type Progress struct {
	// Downloaded is the total number of items downloaded so far.
	Downloaded int64
	// Outstanding is the current size of the input channel for items
	// yet to be downloaded.
	Outstanding int64
}
```
Progress is used to communicate the progress of a download run.


### Type Request
```go
type Request interface {
	Container() file.FS
	FileMode() fs.FileMode // FileMode to use for the downloaded contents.
	Names() []string
}
```
Request represents a request for a list of objects, stored in the same
container, to be downloaded.


### Type Result
```go
type Result struct {
	// Contents of the download, nil on error.
	Contents []byte
	// FileInfo for the downloaded file.
	FileInfo fs.FileInfo
	// Name of the downloaded file.
	Name string
	// Number of retries that were required to download the file.
	Retries int
	// Error encountered during the download.
	Err error
}
```
Result represents the result of the download for a single object.


### Type SimpleRequest
```go
type SimpleRequest struct {
	FS        file.FS
	Filenames []string
	Mode      fs.FileMode
}
```
SimpleRequest is a simple implementation of Request.

### Methods

```go
func (cr SimpleRequest) Container() file.FS
```


```go
func (cr SimpleRequest) FileMode() fs.FileMode
```


```go
func (cr SimpleRequest) Names() []string
```




### Type T
```go
type T interface {
	// Run initiates a download run. It reads Requests from the specified
	// input channel and writes the results of those downloads to the output
	// channel. Closing the input channel indicates to Run that it should
	// complete all outstanding download requests. Run will close the output
	// channel when all requests have been processed.
	Run(ctx context.Context,
		input <-chan Request,
		output chan<- Downloaded) error
}
```
T represents the interface to a downloader that is used to download content.

### Functions

```go
func New(opts ...Option) T
```
New creates a new instance of a download.T.







