# Package [cloudeng.io/aws/s3fs](https://pkg.go.dev/cloudeng.io/aws/s3fs?tab=doc)

```go
import cloudeng.io/aws/s3fs
```

Package s3fs implements fs.FS for AWS S3.

Package s3fs implements fs.FS for AWS S3.

## Functions
### Func DirectoryBucketAZ
```go
func DirectoryBucketAZ(bucket string) string
```

### Func IsDirectoryBucket
```go
func IsDirectoryBucket(bucket string) bool
```

### Func New
```go
func New(cfg aws.Config, options ...Option) filewalk.FS
```
New creates a new instance of filewalk.FS backed by S3.

### Func NewCheckpointOperation
```go
func NewCheckpointOperation(fs *T) checkpoint.Operation
```
NewCheckpointOperation returns a checkpoint.Operation that uses the S3.

### Func NewLevelScanner
```go
func NewLevelScanner(client Client, delimiter byte, path string) filewalk.LevelScanner
```



## Types
### Type Client
```go
type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}
```
Client represents the set of AWS S3 client methods used by s3fs.


### Type Factory
```go
type Factory struct {
	Config  awsconfig.AWSConfig
	Options []Option
}
```
Factory wraps creating an S3FS with the configuration required to correctly
initialize it.

### Methods

```go
func (f Factory) New(ctx context.Context) (*T, error)
```




### Type Option
```go
type Option func(o *options)
```
Option represents an option to New.

### Functions

```go
func WithDelimiter(d byte) Option
```
WithDelimiter sets the delimiter to use when listing objects, the default is
/.


```go
func WithS3Client(client Client) Option
```
WithS3Client specifies the s3.Client to use. If not specified, a new is
created.


```go
func WithS3Options(opts ...func(*s3.Options)) Option
```
WithS3Options wraps s3.Options for use when creating an s3.Client.


```go
func WithScanSize(s int) Option
```
WithScanSize sets the number of items to fetch in a single remote api
invocation for operations such as DeleteAll which may require iterating over
a range of objects.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewS3FS(cfg aws.Config, options ...Option) *T
```
NewS3FS creates a new instance of filewalk.FS and file.ObjectFS backed by
S3.



### Methods

```go
func (s3fs *T) Base(p string) string
```


```go
func (s3fs *T) Delete(ctx context.Context, path string) error
```


```go
func (s3fs *T) DeleteAll(ctx context.Context, path string) error
```


```go
func (s3fs *T) EnsurePrefix(_ context.Context, path string, _ fs.FileMode) error
```


```go
func (s3fs *T) Get(ctx context.Context, path string) ([]byte, error)
```


```go
func (s3fs *T) IsNotExist(err error) bool
```


```go
func (s3fs *T) IsPermissionError(err error) bool
```


```go
func (s3fs *T) Join(components ...string) string
```
Join concatenates the supplied components ensuring to insert delimiters
only when necessary, that is components ending or starting with / (or the
currently configured delimiter) will not


```go
func (fs *T) LevelScanner(prefix string) filewalk.LevelScanner
```


```go
func (s3fs *T) Lstat(ctx context.Context, path string) (file.Info, error)
```


```go
func (s3fs *T) Open(name string) (fs.File, error)
```
Open implements fs.FS.


```go
func (s3fs *T) OpenCtx(ctx context.Context, name string) (fs.File, error)
```
OpenCtx implements file.FS.


```go
func (s3fs *T) Put(ctx context.Context, path string, _ fs.FileMode, data []byte) error
```


```go
func (s3fs *T) Readlink(_ context.Context, _ string) (string, error)
```
ReadLink is not implemented for S3, ie. returns file.ErrNotImplemented.


```go
func (s3fs *T) Scheme() string
```
Scheme implements fs.FS.


```go
func (s3fs *T) Stat(ctx context.Context, name string) (file.Info, error)
```
Stat invokes a Head operation on objects only. If name ends in / (or the
currently configured delimiter) it is considered to be a prefix and a
file.Info is created that reflects that (ie IsDir() returns true).


```go
func (s3fs *T) SysXAttr(existing any, merge file.XAttr) any
```


```go
func (s3fs *T) WriteFile(name string, data []byte, perm fs.FileMode) error
```


```go
func (s3fs *T) WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
```


```go
func (s3fs *T) XAttr(_ context.Context, _ string, info file.Info) (file.XAttr, error)
```







