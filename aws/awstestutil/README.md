# Package [cloudeng.io/aws/awstestutil](https://pkg.go.dev/cloudeng.io/aws/awstestutil?tab=doc)

```go
import cloudeng.io/aws/awstestutil
```

Package awsutil provides support for testing AWS packages and applications.

## Functions
### Func AWSTestMain
```go
func AWSTestMain(m *testing.M, service **AWS, opts ...Option)
```

### Func DefaultAWSConfig
```go
func DefaultAWSConfig() aws.Config
```

### Func SkipAWSTests
```go
func SkipAWSTests(t *testing.T)
```



## Types
### Type AWS
```go
type AWS struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewLocalAWS(opts ...Option) *AWS
```



### Methods

```go
func (a *AWS) KMS(cfg aws.Config) *kms.Client
```


```go
func (a *AWS) S3(cfg aws.Config) *s3.Client
```


```go
func (a *AWS) SecretsManager(cfg aws.Config) *secretsmanager.Client
```


```go
func (a *AWS) Start() error
```


```go
func (a *AWS) Stop() error
```


```go
func (a *AWS) URL() string
```




### Type Option
```go
type Option func(o *Options)
```

### Functions

```go
func WithDebug(log io.Writer) Option
```


```go
func WithKMS() Option
```


```go
func WithS3() Option
```


```go
func WithS3Tree(dir string) Option
```
WithS3Tree configures the local S3 instance with the contents of the
specified directory. The first level of directories under dir are used as
bucket names, the second and deeper levels as prefixes and objects within
those buckets etc.


```go
func WithSecretsManager() Option
```




### Type Options
```go
type Options struct {
	// contains filtered or unexported fields
}
```


### Type Service
```go
type Service string
```

### Constants
### S3, SecretsManager
```go
S3 Service = Service(localstack.S3)
SecretsManager Service = Service(localstack.SecretsManager)

```







