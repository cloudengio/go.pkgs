# Package [cloudeng.io/aws/dbpool](https://pkg.go.dev/cloudeng.io/aws/dbpool?tab=doc)

```go
import cloudeng.io/aws/dbpool
```


## Functions
### Func ConfigWithOverrides
```go
func ConfigWithOverrides(connection string, database, user, host string, port uint16) (*pgxpool.Config, error)
```
ConfigWithOverrides parses the connection string into a pgxpool. Config
and applies any overrides for the database, user, host, or port if they are
non-empty or non-zero.



## Types
### Type Option
```go
type Option func(o *options)
```
Option is a functional option for configuring the connection pool.

### Functions

```go
func WithAWSConfig(cfg aws.Config) Option
```
WithAWSConfig sets the AWS configuration to be used by the TokenGenerator.
The default is to look for the config in the context, but this option allows
it to be explicitly provided.


```go
func WithAcquireConnection(acquire bool) Option
```
WithAcquireConnection forces the pool to acquire a connection during
initialization. This can be used to validate the connection parameters and
fail fast if there are issues.


```go
func WithServerName(serverName string) Option
```
WithServerName sets the TLS ServerName for connections in the pool.
This is required for services like DSQL that use the ServerName for routing
and authentication.


```go
func WithTokenGenerator(tokenGenerator TokenGenerator) Option
```
WithTokenGenerator sets a custom TokenGenerator that will be called to
generate a fresh authentication token for every new connection. This is
essential for services like DSQL that require short-lived tokens.




### Type Pool
```go
type Pool struct {
	*pgxpool.Pool
}
```
Pool is a thin wrapper around pgxpool.Pool that simplifies creating
connection pools.

### Functions

```go
func NewConnectionPool(ctx context.Context, poolConfig *pgxpool.Config, opts ...Option) (*Pool, error)
```




### Type TokenGenerator
```go
type TokenGenerator func(ctx context.Context, cfg aws.Config) (string, error)
```
TokenGenerator is a function type that generates an authentication token.





