# Package [cloudeng.io/aws/awstokens](https://pkg.go.dev/cloudeng.io/aws/awstokens?tab=doc)

```go
import cloudeng.io/aws/awstokens
```

Package awstokens provides a very simple mechanism for retrieving secrets
stored with the AWS secretsmanager service.

## Functions
### Func GetSecret
```go
func GetSecret(ctx context.Context, config aws.Config, nameOrArn string) (string, error)
```
GetSecret returns the value of the secret with the given name or arn.




