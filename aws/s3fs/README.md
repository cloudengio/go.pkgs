# Package [cloudeng.io/aws/s3fs](https://pkg.go.dev/cloudeng.io/aws/s3fs?tab=doc)

```go
import cloudeng.io/aws/s3fs
```

Package s3fs implements fs.FS for AWS S3.

Package s3fs implements fs.FS for AWS S3.

## Functions
### Func New
```go
func New(cfg aws.Config, options ...Option) file.FS
```
New creates a new instance of file.FS backed by S3.



## Types
### Type Client
```go
type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}
```
Client represents the set of AWS S3 client methods used by s3fs.


### Type Factory
```go
type Factory struct {
	Config awsconfig.AWSFlags
}
```
Factory implements file.FSFactory for AWS S3.

### Methods

```go
func (d Factory) New(ctx context.Context, scheme string) (file.FS, error)
```
New implements file.FSFactory.


```go
func (d Factory) NewFromMatch(ctx context.Context, match cloudpath.Match) (file.FS, error)
```




### Type Option
```go
type Option func(o *options)
```
Option represents an option to New.

### Functions

```go
func WithS3Client(client Client) Option
```
WithS3Client specifies the s3.Client to use. If not specified, a new is
created.


```go
func WithS3Options(opts ...func(*s3.Options)) Option
```
WithS3Options wraps s3.Options for use when creating an s3.Client.







