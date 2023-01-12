# Package [cloudeng.io/aws/s3fs](https://pkg.go.dev/cloudeng.io/aws/s3fs?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/aws/s3fs)](https://goreportcard.com/report/cloudeng.io/aws/s3fs)

```go
import cloudeng.io/aws/s3fs
```

Package s3fs implements fs.FS for AWS S3.

## Functions
### Func New
```go
func New(cfg aws.Config, options ...Option) file.FS
```



## Types
### Type Client
```go
type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}
```


### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithS3Client(client Client) Option
```


```go
func WithS3Options(opts ...func(*s3.Options)) Option
```







