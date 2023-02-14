# Package [cloudeng.io/aws/s3fs/s3fstestutil](https://pkg.go.dev/cloudeng.io/aws/s3fs/s3fstestutil?tab=doc)

```go
import cloudeng.io/aws/s3fs/s3fstestutil
```


## Functions
### Func NewMockFS
```go
func NewMockFS(fs file.FS, opts ...Option) s3fs.Client
```



## Types
### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithBucket(b string) Option
```


```go
func WithLeadingSlashStripped() Option
```







