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
	Name          string           `yaml:"name"`
	Depth         int              `yaml:"depth"`
	Seeds         []string         `yaml:"seeds"`
	NoFollowRules []string         `yaml:"nofollow"`
	FollowRules   []string         `yaml:"follow"`
	RewriteRules  []string         `yaml:"rewrite"`
	Download      DownloadConfig   `yaml:"download"`
	NumExtractors int              `yaml:"num_extractors"`
	Extractors    []content.Type   `yaml:"extractors"`
	Cache         CrawlCacheConfig `yaml:"cache"`
}
```
Config represents the configuration for a single crawl.

### Methods

```go
func (c Config) CreateSeedCrawlRequests(ctx context.Context, factories map[string]file.FSFactory, seeds map[string][]cloudpath.Match) ([]download.Request, error)
```
CreateSeedCrawlRequests creates a set of crawl requests for the supplied
seeds. It use the factories to create a file.FS for the URI scheme of each
seed.


```go
func (c Config) ExtractorRegistry(avail map[content.Type]outlinks.Extractor) (*content.Registry[outlinks.Extractor], error)
```
ExtractorRegistry returns a content.Registry containing for
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
	Prefix            string `yaml:"cache_prefix"`
	ClearBeforeCrawl  bool   `yaml:"cache_clear_before_crawl"`
	Checkpoint        string `yaml:"cache_checkpoint"`
	ShardingPrefixLen int    `yaml:"cache_sharding_prefix_len"`
}
```
Each crawl may specify its own cache directory and configuration. This
will be used to store the results of the crawl. The cache is intended to be
relative to the

### Methods

```go
func (c CrawlCacheConfig) Initialize(root string) (cachePath, checkpointPath string, err error)
```
Initialize creates the cache and checkpoint directories relative to
the specified root, and optionally clears them before the crawl (if
Cache.ClearBeforeCrawl is true). Any environment variables in the root or
Cache.Prefix will be expanded.




### Type Crawler
```go
type Crawler struct {
	Config
	Extractors func() map[content.Type]outlinks.Extractor
	// contains filtered or unexported fields
}
```
Crawler represents a crawler instance and contains global configuration
information.

### Methods

```go
func (c *Crawler) Run(ctx context.Context, fsMap map[string]file.FSFactory, cacheRoot string, displayOutlinks, displayProgress bool) error
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
	DefaultConcurrency       int   `yaml:"default_concurrency"`
	DefaultRequestChanSize   int   `yaml:"default_request_chan_size"`
	DefaultCrawledChanSize   int   `yaml:"default_crawled_chan_size"`
	PerDepthConcurrency      []int `yaml:"per_depth_concurrency"`
	PerDepthRequestChanSizes []int `yaml:"per_depth_request_chan_sizes"`
	PerDepthCrawledChanSizes []int `yaml:"per_depth_crawled_chan_sizes"`
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
	InitialDelay time.Duration `yaml:"initial_delay"`
	Steps        int           `yaml:"steps"`
	StatusCodes  []int         `yaml:"status_codes,flow"`
}
```
ExponentialBackoffConfig is the configuration for an exponential backoff
retry strategy for downloads.


### Type Rate
```go
type Rate struct {
	Tick            time.Duration `yaml:"tick"`
	RequestsPerTick int           `yaml:"requests_per_tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick"`
}
```
Rate specifies a rate in one of several forms, only one should be used.


### Type RateControl
```go
type RateControl struct {
	Rate               Rate               `yaml:"rate_control"`
	ExponentialBackoff ExponentialBackoff `yaml:"exponential_backoff"`
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







