# Package [cloudeng.io/file/cachefs](https://pkg.go.dev/cloudeng.io/file/cachefs?tab=doc)

```go
import cloudeng.io/file/cachefs
```

Package cachefs provides a caching layer for ReadFileFS implementations.

## Constants
### DefaultTTL, DefaultCleanupInterval
```go
DefaultTTL = 24 * time.Hour
DefaultCleanupInterval = 1 * time.Hour

```



## Types
### Type CachingReadfileFS
```go
type CachingReadfileFS struct {
	// contains filtered or unexported fields
}
```
CachingReadfileFS implements a caching layer over a ReadFileFS.

### Functions

```go
func NewCachingReadFileFS(fs file.ReadFileFS, opts ...Option) *CachingReadfileFS
```
NewCachingReadFileFS creates a new CachingReadfileFS with the specified TTL
and cleanup interval. It starts a background goroutine to periodically clear
out expired cache entries. Call Close to stop the background goroutine.



### Methods

```go
func (c *CachingReadfileFS) Close() error
```
Close stops the background cleanup goroutine.


```go
func (c *CachingReadfileFS) Invalidate(name string)
```
Invalidate removes the named file from the cache.


```go
func (c *CachingReadfileFS) ReadFile(name string) ([]byte, error)
```
ReadFile reads the named file, utilizing the cache if the entry is fresh.


```go
func (c *CachingReadfileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```
ReadFileCtx reads the named file using the provided context, utilizing the
cache if fresh.




### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithCleanupInterval(d time.Duration) Option
```
WithCleanupInterval specifies the interval for periodic cleanup of expired
cache entries. The default is 1 minute.


```go
func WithTTL(d time.Duration) Option
```
WithTTL specifies the time-to-live for cache entries. The default is 5
minutes.







