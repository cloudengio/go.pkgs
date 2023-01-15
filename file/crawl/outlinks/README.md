# Package [cloudeng.io/file/crawl/outlinks](https://pkg.go.dev/cloudeng.io/file/crawl/outlinks?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/crawl/outlinks)](https://goreportcard.com/report/cloudeng.io/file/crawl/outlinks)

```go
import cloudeng.io/file/crawl/outlinks
```


## Functions
### Func NewExtractors
```go
func NewExtractors(errCh chan<- Errors, extractors ...Extractor) crawl.Outlinks
```
NewExtractors creates a crawl.Outlinks.Extractor given instances of the
lower level Extractor interface. The extractors are run in turn until one
returns a set



## Types
### Type Download
```go
type Download struct {
	Request   download.Request
	Container file.FS
	Download  download.Result
}
```
Download represents a single downloaded file, as opposed to
download.Downloaded which represents multiple files in the same container.
It's a convenience for use by the Extractor interface.


### Type ErrorDetail
```go
type ErrorDetail struct {
	download.Result
	Error error
}
```


### Type Errors
```go
type Errors struct {
	Request   download.Request
	Container file.FS
	Errors    []ErrorDetail
}
```


### Type Extractor
```go
type Extractor interface {
	// MimeType returns the mime type that this extractor is capable of handling.
	MimeType() string
	// Outlinks extracts outlinks from the specified downloaded file.
	Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
	Request(depth int, download Download, outlinks []string) download.Request
}
```
Extractor is a lower level interface for outlink extractors that allows for
the separation of extracting outlinks and creating new download requests to
retrieve them. This allows for easier customization of the crawl process,
for example, to rewrite or otherwise manipulate the link names.


### Type HTML
```go
type HTML struct {
	// contains filtered or unexported fields
}
```
HTML is an outlink extractor for HTML documents. It implements both
crawl.Outlinks and outlinks.Extractor.

### Functions

```go
func NewHTML() *HTML
```



### Methods

```go
func (ho *HTML) HREFs(rd io.Reader) ([]string, error)
```
HREFs returns the hrefs found in the provided HTML document.


```go
func (ho *HTML) IsDup(link string) bool
```
IsDup returns true if link has been seen before (ie. has been used as an
argument to IsDup).


```go
func (ho *HTML) MimeType() string
```


```go
func (ho *HTML) Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
```
Outlinks implements Extractor.Outlinks.


```go
func (ho *HTML) Request(depth int, download Download, outlinks []string) download.Request
```
Request implements Extractor.Request.







