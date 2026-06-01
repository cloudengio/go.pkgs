# Package [cloudeng.io/aws/awsutil](https://pkg.go.dev/cloudeng.io/aws/awsutil?tab=doc)

```go
import cloudeng.io/aws/awsutil
```


## Functions
### Func AccountID
```go
func AccountID(ctx context.Context, cfg aws.Config) (string, error)
```
AccountID returns the account id from the aws.Config and caches it locally.

### Func InterpretError
```go
func InterpretError(err error) error
```
InterpretError attempts to interpret AWS SDK errors and either improve the
error reporting to the caller and/or map to already defined error types as
fs.ErrNotExist.

secretmanager.ResourceNotFoundException is mapped to fs.ErrNotExist, and
secretmanager.InvalidRequestException with "currently marked deleted" in the
message is also mapped to fs.ErrNotExist, as the secret is not accessible.

The error message "security token included in the request is invalid" can
be caused by multiple issues, such as an incorrect Secret Access Key,
an expired Session Token (very common with IAM roles/temporary credentials),
or an incorrect Access Key ID. This is interpreted and the returned error is
wrapped with a hint to check AWS credentials/configuration.

### Func IsARN
```go
func IsARN(name string) bool
```
IsArn returns true if the supplied string is an ARN.

### Func Region
```go
func Region(_ context.Context, cfg aws.Config) string
```
Region obtains the AWS region either from the supplied config or from the
environment.




