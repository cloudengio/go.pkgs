# Package [cloudeng.io/file/cachefs](https://pkg.go.dev/cloudeng.io/file/cachefs?tab=doc)

```go
import cloudeng.io/file/cachefs
```

Package cachefs provides a caching and related wrappers for ReadFileFS
implementations.

## Constants
### DefaultTTL, DefaultCleanupInterval, DefaultSingleFlight
```go
DefaultTTL = 24 * time.Hour
DefaultCleanupInterval = 1 * time.Hour
DefaultSingleFlight = false

```



## Types
### Type CachingReadFileFS
```go
type CachingReadFileFS struct {
	// contains filtered or unexported fields
}
```
CachingReadFileFS implements a caching layer over a ReadFileFS that is
suitable for a small numbers of small files that can be readily kept in
memory.

### Functions

```go
func NewCachingReadFileFS(fs file.ReadFileFS, opts ...Option) *CachingReadFileFS
```
NewCachingReadFileFS creates a new CachingReadFileFS with the specified TTL
and cleanup interval. It starts a background goroutine to periodically clear
out expired cache entries. Call Close to stop the background goroutine.



### Methods

```go
func (c *CachingReadFileFS) Forget(name string)
```
Forget removes the named file from the cache.


```go
func (c *CachingReadFileFS) ReadFile(name string) ([]byte, error)
```
ReadFile reads the named file, utilizing the cache if the entry is fresh.


```go
func (c *CachingReadFileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```
ReadFileCtx reads the named file using the provided context, utilizing the
cache if fresh.


```go
func (c *CachingReadFileFS) Stop(ctx context.Context) error
```
Stop stops the background cleanup goroutine.




### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithCleanupInterval(d time.Duration) Option
```
WithCleanupInterval specifies the interval for periodic background cleanup
of expired entries. The default is DefaultCleanupInterval. A value of 0
disables periodic cleanup, with expired entries being overwritten on access.


```go
func WithSingleFlight(v bool) Option
```
WithSingleFlight enables single-flight behavior for concurrent calls to
ReadFileCtx with the same name. The default is false.


```go
func WithTTL(d time.Duration) Option
```
WithTTL specifies the time-to-live for cache entries. The default is
DefaultTTL.




### Type SingleFlightReadFileFS
```go
type SingleFlightReadFileFS struct {
	// contains filtered or unexported fields
}
```
SingleFlightReadFileFS is a wrapper around a ReadFileFS that provides
single-flight behavior for concurrent calls to ReadFileCtx and ReadFile with
the same name. This can be used in conjunction with CachingReadFileFS to
prevent thundering herd issues on cache misses.

### Functions

```go
func NewSingleFlightReadFileFS(fs file.ReadFileFS) *SingleFlightReadFileFS
```



### Methods

```go
func (s *SingleFlightReadFileFS) ReadFile(name string) ([]byte, error)
```


```go
func (s *SingleFlightReadFileFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```







