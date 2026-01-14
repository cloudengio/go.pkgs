# Package [cloudeng.io/file/crawl/crawlcmd](https://pkg.go.dev/cloudeng.io/file/crawl/crawlcmd?tab=doc)

```go
import cloudeng.io/file/crawl/crawlcmd
```

Package crawlcmd provides support for building command line tools for
crawling. In particular it provides support for managing the configuration
of a crawl via yaml.

## Types
### Type Config
```go
type Config struct {
	Name          string           `yaml:"name" doc:"the name of the crawl"`
	Depth         int              `yaml:"depth" doc:"the maximum depth to crawl"`
	Seeds         []string         `yaml:"seeds" doc:"the initial set of URIs to crawl"`
	NoFollowRules []string         `yaml:"nofollow" doc:"a set of regular expressions that will be used to determine which links to not follow. The regular expressions are applied to the full URL."`
	FollowRules   []string         `yaml:"follow" doc:"a set of regular expressions that will be used to determine which links to follow. The regular expressions are applied to the full URL."`
	RewriteRules  []string         `yaml:"rewrite" doc:"a set of regular expressions that will be used to rewrite links. The regular expressions are applied to the full URL."`
	Download      DownloadConfig   `yaml:"download" doc:"the configuration for downloading documents"`
	NumExtractors int              `yaml:"num_extractors" doc:"the number of concurrent link extractors to use"`
	Extractors    []content.Type   `yaml:"extractors" doc:"the content types to extract links from"`
	Cache         CrawlCacheConfig `yaml:"cache" doc:"the configuration for the cache of downloaded documents"`
}
```
Config represents the configuration for a single crawl.

### Methods

```go
func (c Config) CreateSeedCrawlRequests(
	ctx context.Context,
	factories map[string]FSFactory,
	seeds map[string][]cloudpath.Match,
) ([]download.Request, error)
```
CreateSeedCrawlRequests creates a set of crawl requests for the supplied
seeds. It use the factories to create a file.FS for the URI scheme of each
seed.


```go
func (c Config) ExtractorRegistry(avail map[content.Type]outlinks.Extractor) (*content.Registry[outlinks.Extractor], error)
```
ExtractorRegistry returns a content.Registry containing the
outlinks.Extractor that can be used with outlinks.Extract.


```go
func (c Config) NewLinkProcessor() (*outlinks.RegexpProcessor, error)
```
NewLinkProcessor creates a outlinks.RegexpProcessor using the nofollow,
follow and reqwrite specifications in the configuration.


```go
func (c Config) SeedsByScheme(matchers cloudpath.MatcherSpec) (map[string][]cloudpath.Match, []string)
```
SeedsByScheme returns the crawl seeds grouped by their scheme and any seeds
that are not recognised by the supplied cloudpath.MatcherSpec.




### Type CrawlCacheConfig
```go
type CrawlCacheConfig struct {
	Downloads         string    `yaml:"downloads" doc:"the prefix/directory to use for the cache of downloaded documents. This is an absolute path the root directory of the crawl."`
	ClearBeforeCrawl  bool      `yaml:"clear_before_crawl" doc:"if true, the cache and checkpoint will be cleared before the crawl starts."`
	Checkpoint        string    `yaml:"checkpoint" doc:"the location of any checkpoint data used to resume a crawl, this is an absolute path."`
	ShardingPrefixLen int       `yaml:"sharding_prefix_len" doc:"the number of characters of the filename to use for sharding the cache. This is intended to avoid filesystem limits on the number of files in a directory."`
	Concurrency       int       `yaml:"concurrency" doc:"the number of concurrent operations to use when reading/writing to the cache."`
	ServiceConfig     yaml.Node `yaml:"service_config,omitempty" doc:"cache service specific configuration, eg. AWS specific configuration"`
}
```
Each crawl may specify its own cache directory and configuration. This
will be used to store the results of the crawl. The ServiceSpecific field
is intended to be parametized to some service specific configuration for
cache services that require it, such as AWS S3. This is deliberately left to
client packages to avoid depenedency bloat in core packages such as this.
The type of the ServiceConfig file is generally determined using the
scheme of the Downloads path (e.g s3://... would imply an AWS specific
configuration).

### Methods

```go
func (c CrawlCacheConfig) CheckpointPath() string
```
CheckpointPath returns the expanded checkpoint path.


```go
func (c CrawlCacheConfig) DownloadPath() string
```
DownloadPath returns the expanded downloads path.


```go
func (c CrawlCacheConfig) PrepareCheckpoint(ctx context.Context, op checkpoint.Operation) error
```
PrepareCheckpoint initializes the checkpoint operation (ie. calls
op.Init(ctx, checkpointPath)) and optionally clears the checkpoint if
ClearBeforeCrawl is true. It returns an error if the checkpoint cannot be
initialized or cleared.


```go
func (c CrawlCacheConfig) PrepareDownloads(ctx context.Context, fs content.FS) error
```
PrepareDownloads ensures that the cache directory exists and is empty if
ClearBeforeCrawl is true. It returns an error if the directory cannot be
created or cleared.




### Type Crawler
```go
type Crawler struct {
	// contains filtered or unexported fields
}
```
Crawler represents a crawler instance and contains global configuration
information.

### Functions

```go
func NewCrawler(cfg Config, resources Resources) *Crawler
```
NewCrawler creates a new crawler instance using the supplied configuration
and resources.



### Methods

```go
func (c *Crawler) Run(ctx context.Context,
	displayOutlinks, displayProgress bool) error
```
Run runs the crawler.




### Type DownloadConfig
```go
type DownloadConfig struct {
	DownloadFactoryConfig `yaml:",inline"`
	RateControlConfig     RateControl `yaml:",inline"`
}
```


### Type DownloadFactoryConfig
```go
type DownloadFactoryConfig struct {
	DefaultConcurrency       int   `yaml:"default_concurrency" doc:"the number of concurrent downloads (defaults to GOMAXPROCS(0)), used when a per crawl depth value is not specified via per_depth_concurrency."`
	DefaultRequestChanSize   int   `yaml:"default_request_chan_size" doc:"the size of the channel used to queue download requests, used when a per crawl depth value is not specified via per_depth_request_chan_sizes. Increased values allow for more concurrency between discovering new items to crawl and crawling them."`
	DefaultCrawledChanSize   int   `yaml:"default_crawled_chan_size" doc:"the size of the channel used to queue downloaded items, used when a per crawl depth value is not specified via per_depth_crawled_chan_sizes. Increased values allow for more concurrency between downloading documents and processing them."`
	PerDepthConcurrency      []int `yaml:"per_depth_concurrency" doc:"per crawl depth values for the number of concurrent downloads"`
	PerDepthRequestChanSizes []int `yaml:"per_depth_request_chan_sizes" doc:"per crawl depth values for the size of the channel used to queue download requests"`
	PerDepthCrawledChanSizes []int `yaml:"per_depth_crawled_chan_sizes" doc:"per crawl depth values for the size of the channel used to queue downloaded items"`
}
```
DownloadFactoryConfig is the configuration for a crawl.DownloaderFactory.

### Methods

```go
func (df DownloadFactoryConfig) Depth0Chans() (chan download.Request, chan crawl.Crawled)
```
Depth0Chans creates the chanels required to start the crawl with their
capacities set to the values specified in the DownloadFactoryConfig for a
depth0 crawl, or the default values if none are specified.


```go
func (df DownloadFactoryConfig) NewFactory(ch chan<- download.Progress) crawl.DownloaderFactory
```
NewFactory returns a new instance of a crawl.DownloaderFactory which is
parametised via its DownloadFactoryConfig receiver.




### Type ExponentialBackoff
```go
type ExponentialBackoff struct {
	InitialDelay time.Duration `yaml:"initial_delay" doc:"the initial delay between retries for exponential backoff"`
	Steps        int           `yaml:"steps" doc:"the number of steps of exponential backoff before giving up"`
	StatusCodes  []int         `yaml:"status_codes,flow" doc:"the status codes that trigger a retry"`
}
```
ExponentialBackoffConfig is the configuration for an exponential backoff
retry strategy for downloads.


### Type FSFactory
```go
type FSFactory func(context.Context) (file.FS, error)
```
FSFactory is a function that returns a file.FS used to crawl a given FS.


### Type Rate
```go
type Rate struct {
	Tick            time.Duration `yaml:"tick" doc:"the duration of a tick"`
	RequestsPerTick int           `yaml:"requests_per_tick" doc:"the number of requests per tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick" doc:"the number of bytes per tick"`
}
```
Rate specifies a rate in one of several forms, only one should be used.


### Type RateControl
```go
type RateControl struct {
	Rate               Rate               `yaml:"rate_control" doc:"the rate control parameters"`
	ExponentialBackoff ExponentialBackoff `yaml:"exponential_backoff" doc:"the exponential backoff parameters"`
}
```
RateControl is the configuration for rate based control of download
requests.

### Methods

```go
func (c RateControl) NewRateController() (*ratecontrol.Controller, error)
```
NewRateController creates a new rate controller based on the values
contained in RateControl.




### Type Resources
```go
type Resources struct {
	// Extractors are used to extract outlinks from crawled documents
	// based on their content type.
	Extractors map[content.Type]outlinks.Extractor
	// CrawlStoreFactories are used to create file.FS instances for
	// the files being crawled based on their scheme.
	CrawlStoreFactories map[string]FSFactory
	// ContentStoreFactory is a function that returns a content.FS used to store
	// the downloaded content.
	NewContentFS func(context.Context, CrawlCacheConfig) (content.FS, error)
}
```
Resources contains the resources required by the crawler.




## Examples
### [ExampleCrawlCacheConfig](https://pkg.go.dev/cloudeng.io/file/crawl/crawlcmd?tab=doc#example-CrawlCacheConfig)




