# Package [cloudeng.io/aws/rds](https://pkg.go.dev/cloudeng.io/aws/rds?tab=doc)

```go
import cloudeng.io/aws/rds
```


## Functions
### Func GenerateDSQLToken
```go
func GenerateDSQLToken(ctx context.Context, endpoint string, cfg aws.Config) (string, error)
```
GenerateDSQLToken creates a 15-minute SigV4 signed authentication token.

### Func TokenGenerator
```go
func TokenGenerator(endpoint string) dbpool.TokenGenerator
```
TokenGenerator returns a dbpool.TokenGenerator that generates DSQL
authentication tokens.




