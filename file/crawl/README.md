# Package [cloudeng.io/file/crawl](https://pkg.go.dev/cloudeng.io/file/crawl?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/file/crawl)](https://goreportcard.com/report/cloudeng.io/file/crawl)

```go
import cloudeng.io/file/crawl
```

Package crawl provides a framework for multilevel/recursive crawling files.
As files are downloaded, they may be processed by an outlinks extractor
which yields more files to crawled. Typically such a multilevel crawl is
limited to a set number of iterations referred to as the depth of the crawl.
The interface to a crawler is channel based to allow for concurrency. The
outlink extractor is called for all downloaded files and should implement
duplicate detection and removal.

## Types
### Type Crawled
```go
type Crawled struct {
	download.Downloaded
	Outlinks []download.Request
	Depth    int // The depth at which the document was crawled.
}
```
Crawled represents all of the downloaded content in response to a given
crawl request.


### Type DownloaderFactory
```go
type DownloaderFactory func(ctx context.Context, depth int) (
	downloader download.T,
	input chan download.Request,
	output chan download.Downloaded)
```
DownloaderFactory is used to create a new downloader for each 'depth' in
a multilevel crawl. The depth argument can be used to create different
configurations of the downloader tailored to the depth of the crawl.
For example, lower depths would use less concurrency in the downloader
since there are very likely fewer files to be downloaded than at higher ones
(since more links will have extracted).


### Type Option
```go
type Option func(o *options)
```
Option is used to configure the behaviour of a newly created Crawler.

### Functions

```go
func WithCrawlDepth(depth int) Option
```
WithCrawlDepth sets the depth of the crawl.


```go
func WithNumExtractors(concurrency int) Option
```
WithNumExtractors sets the number of extractors to run.




### Type Outlinks
```go
type Outlinks interface {
	// Note that the implementation of Extract is responsible for removing
	// duplicates from the set of extracted links returned.
	Extract(ctx context.Context, depth int, download download.Downloaded) []download.Request
}
```
Outlinks is the interface to an 'outlink' extractor, that is, an entity that
determines additional items to be downloaded based on the contents of an
already downloaded one.


### Type SimpleRequest
```go
type SimpleRequest struct {
	download.SimpleRequest
	Depth int
}
```
SimpleRequest is a simple implementation of download.Request with an
additional field to record the depth that the request was created at.
This will typically be set by an outlink extractor.


### Type T
```go
type T interface {
	Run(ctx context.Context,
		factory DownloaderFactory,
		extractor Outlinks,
		writeFS file.WriteFS,
		input <-chan download.Request,
		output chan<- Crawled) error
}
```
T represents the interface to a crawler.

### Functions

```go
func New(opts ...Option) T
```
New creates a new instance of T that implements a multilevel, concurrent
crawl. The crawl is implemented as a chain of downloaders and extractors,
one per depth requested. This allows for concurrency within each level of
the crawl as well as across each level.







