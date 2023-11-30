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




