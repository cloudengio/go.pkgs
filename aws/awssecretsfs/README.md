# Package [cloudeng.io/aws/awssecretsfs](https://pkg.go.dev/cloudeng.io/aws/awssecretsfs?tab=doc)

```go
import cloudeng.io/aws/awssecretsfs
```

Package awssecrets provides an implementation of fs.ReadFileFS that reads
secrets from the AWS secretsmanager.

## Functions
### Func New
```go
func New(cfg aws.Config, options ...Option) fs.ReadFileFS
```
New creates a new instance of fs.ReadFile backed by the secretsmanager.



## Types
### Type Client
```go
type Client interface {
	ListSecretVersionIds(ctx context.Context, params *secretsmanager.ListSecretVersionIdsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretVersionIdsOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}
```
Client represents the set of AWS S3 client methods used by s3fs.


### Type Option
```go
type Option func(o *options)
```
Option represents an option to New.

### Functions

```go
func WithSecretsClient(client Client) Option
```
WithSecretsClient specifies the secretsmanager.Client to use. If not
specified, a new is created.


```go
func WithSecretsOptions(opts ...func(*secretsmanager.Options)) Option
```
WithSecretsOptions wraps secretsmanager.Options for use when creating an
s3.Client.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T implements fs.ReadFileFS for secretsmanager.

### Functions

```go
func NewSecretsFS(cfg aws.Config, options ...Option) *T
```
NewSecretsFS creates a new instance of T.



### Methods

```go
func (smfs *T) Open(name string) (fs.File, error)
```
Open implements fs.FS. Name can be the short name of the secret or the ARN.


```go
func (smfs *T) ReadFile(name string) ([]byte, error)
```
ReadFile implements fs.ReadFileFS. Name can be the short name of the secret
or the ARN.


```go
func (smfs *T) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```
ReadFileCtx is like ReadFile but with a context.







