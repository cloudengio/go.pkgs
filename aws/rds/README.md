# Package [cloudeng.io/aws/rds](https://pkg.go.dev/cloudeng.io/aws/rds?tab=doc)

```go
import cloudeng.io/aws/rds
```


## Functions
### Func GenerateDSQLToken
```go
func GenerateDSQLToken(ctx context.Context, endpoint string, admin bool, cfg aws.Config, opts ...func(*auth.TokenOptions)) (string, error)
```
GenerateDSQLToken creates a 15-minute SigV4 signed authentication token.

### Func TokenGenerator
```go
func TokenGenerator(endpoint string, admin bool, opts ...func(*auth.TokenOptions)) dbpool.TokenGenerator
```
TokenGenerator returns a dbpool.TokenGenerator that generates DSQL
authentication tokens.

### Func WithDSQLTokenExpiration
```go
func WithDSQLTokenExpiration(expiration time.Duration) func(o *auth.TokenOptions)
```
WithDSQLTokenExpiration returns a function that can be passed to
GenerateDSQLToken to set the expiration time of the generated token.




