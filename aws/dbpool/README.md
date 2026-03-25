# Package [cloudeng.io/aws/dbpool](https://pkg.go.dev/cloudeng.io/aws/dbpool?tab=doc)

```go
import cloudeng.io/aws/dbpool
```


## Types
### Type Option
```go
type Option func(o *options)
```
Option is a functional option for configuring the connection pool.

### Functions

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
func NewConnectionPool(ctx context.Context, connection string, opts ...Option) (*Pool, error)
```




### Type TokenGenerator
```go
type TokenGenerator func(ctx context.Context) (string, error)
```
TokenGenerator is a function type that generates an authentication token.





