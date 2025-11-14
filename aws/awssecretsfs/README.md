# Package [cloudeng.io/aws/awssecretsfs](https://pkg.go.dev/cloudeng.io/aws/awssecretsfs?tab=doc)

```go
import cloudeng.io/aws/awssecretsfs
```

Package awssecrets provides an implementation of fs.ReadFileFS that reads
secrets from the AWS secretsmanager.

## Types
### Type Client
```go
type Client interface {
	ListSecretVersionIds(ctx context.Context, params *secretsmanager.ListSecretVersionIdsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretVersionIdsOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
}
```
Client represents the set of AWS Secrets service methods used by
awssecretsfs.


### Type Option
```go
type Option func(o *options)
```
Option represents an option to New.

### Functions

```go
func WithAllowCreation(allow bool) Option
```
WithAllowCreation specifies whether creation of new secrets is allowed.


```go
func WithAllowUpdates(allow bool) Option
```
WithAllowUpdates specifies whether writes to existing secrets are allowed.


```go
func WithRecoveryDelay(days int64) Option
```
WithRecoveryDelay specifies the number of days to retain a secret after
deletion. Set to 0 for immediate deletion without recovery, the default is 7
days.


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
func New(cfg aws.Config, options ...Option) *T
```
New creates a new instance of fs.ReadFile backed by the secretsmanager.


```go
func NewSecretsFS(cfg aws.Config, options ...Option) *T
```
NewSecretsFS creates a new instance of T.



### Methods

```go
func (smfs *T) Delete(ctx context.Context, nameOrArn string) error
```
Delete deletes the secret with the given name. Name can be the short name of
the secret or the ARN.


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
func (smfs *T) ReadFileCtx(ctx context.Context, nameOrArn string) ([]byte, error)
```
ReadFileCtx is like ReadFile but with a context.


```go
func (smfs *T) WriteFileCtx(ctx context.Context, nameOrArn string, data []byte, _ fs.FileMode) error
```







