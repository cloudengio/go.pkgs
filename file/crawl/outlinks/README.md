# Package [cloudeng.io/file/crawl/outlinks](https://pkg.go.dev/cloudeng.io/file/crawl/outlinks?tab=doc)

```go
import cloudeng.io/file/crawl/outlinks
```


## Functions
### Func NewExtractors
```go
func NewExtractors(errCh chan<- Errors, processor Process, extractors *content.Registry[Extractor]) crawl.Outlinks
```
NewExtractors creates a crawl.Outlinks.Extractor given instances of the
lower level Extractor interface. The extractors that match the downloaded
content's mime type are run for that content.



## Types
### Type Download
```go
type Download struct {
	Request  download.Request
	Download download.Result
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

### Methods

```go
func (e Errors) String() string
```




### Type Extractor
```go
type Extractor interface {
	// ContentType returns the mime type that this extractor is capable of handling.
	ContentType() content.Type
	// Outlinks extracts outlinks from the specified downloaded file. This
	// is generally specific to the mime type of the content being processed.
	Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
	// Request creates new download requests for the specified outlinks.
	Request(depth int, download Download, outlinks []string) download.Request
}
```
Extractor is a lower level interface for outlink extractors that allows for
the separation of extracting outlinks, filtering/rewriting them and creating
new download requests to retrieve them. This allows for easier customization
of the crawl process, for example, to rewrite or otherwise manipulate the
link names or create appropriate crawl requests for different types of
outlink.


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
func (ho *HTML) ContentType() content.Type
```


```go
func (ho *HTML) HREFs(base string, rd io.Reader) ([]string, error)
```
HREFs returns the hrefs found in the provided HTML document.


```go
func (ho *HTML) IsDup(link string) bool
```
IsDup returns true if link has been seen before (ie. has been used as an
argument to IsDup).


```go
func (ho *HTML) Outlinks(_ context.Context, _ int, download Download, contents io.Reader) ([]string, error)
```
Outlinks implements Extractor.Outlinks.


```go
func (ho *HTML) Request(depth int, download Download, outlinks []string) download.Request
```
Request implements Extractor.Request.




### Type PassthroughProcessor
```go
type PassthroughProcessor struct{}
```
PassthroughProcessor implements Process and simply returns its input.

### Methods

```go
func (pp *PassthroughProcessor) Process(outlinks []string) []string
```




### Type Process
```go
type Process interface {
	Process(outlink []string) []string
}
```
Process is an interface for processing outlinks.


### Type RegexpProcessor
```go
type RegexpProcessor struct {
	NoFollow []string // regular expressions that match links that should be ignored.
	Follow   []string // regular expressions that match links that should be followed. Follow overrides NoFollow.
	Rewrite  []string // rewrite rules that are applied to links that are followed specified as textutil.RewriteRule strings
	// contains filtered or unexported fields
}
```
RegexpProcessor is an implementation of Process that uses regular
expressions to determine whether a link should be ignored (nofollow),
followed or rewritten. Follow overrides nofollow and only links that make
it through both nofollow and follow are rewritten. Each of the rewrites is
applied in turn and all of the rewritten values are returned.

### Methods

```go
func (cfg *RegexpProcessor) Compile() error
```
Compile is called to compile all of the regular expressions contained within
the processor. It must be called before Process.


```go
func (cfg *RegexpProcessor) Process(outlinks []string) []string
```







