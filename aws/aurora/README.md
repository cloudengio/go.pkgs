# Package [cloudeng.io/aws/aurora](https://pkg.go.dev/cloudeng.io/aws/aurora?tab=doc)

```go
import cloudeng.io/aws/aurora
```


## Functions
### Func GenerateDSQLToken
```go
func GenerateDSQLToken(ctx context.Context, endpoint string) (string, error)
```
GenerateDSQLToken creates a 15-minute SigV4 signed authentication token.

### Func TokenGenerator
```go
func TokenGenerator(endpoint string) dbpool.TokenGenerator
```
TokenGenerator returns a dbpool.TokenGenerator that generates DSQL
authentication tokens.




