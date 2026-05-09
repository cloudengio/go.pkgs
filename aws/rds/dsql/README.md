# Package [cloudeng.io/aws/rds/dsql](https://pkg.go.dev/cloudeng.io/aws/rds/dsql?tab=doc)

```go
import cloudeng.io/aws/rds/dsql
```

Package dsql provides utilities for working with AWS DSQL, including
generating authentication tokens and managing DSQL-related VPC endpoints.

## Functions
### Func GenerateToken
```go
func GenerateToken(ctx context.Context, endpoint string, admin bool, cfg aws.Config, opts ...func(*auth.TokenOptions)) (string, error)
```
GenerateToken creates a 15-minute SigV4 signed authentication token.

### Func PrivateLinkServiceName
```go
func PrivateLinkServiceName(ctx context.Context, clusterID string) (publicEndpoint, endpointServiceName string, err error)
```
PrivateLinkServiceName is a helper function that retrieves the VPC public
endpoint and endpoint service name for a DSQL cluster given its cluster
ID or endpoint hostname. service name for a given cluster ID using the
aws.Config from the context to create a DSQL client. This is a convenient
wrapper around Cluster.GetPrivateLinkServiceName. clusterID may be
either a bare 26-character cluster ID or a full endpoint hostname (e.g.
"<id>.dsql.us-east-1.on.aws").

### Func TokenGenerator
```go
func TokenGenerator(endpoint string, admin bool, opts ...func(*auth.TokenOptions)) dbpool.TokenGenerator
```
TokenGenerator returns a dbpool.TokenGenerator that generates DSQL
authentication tokens.

### Func WithTokenExpiration
```go
func WithTokenExpiration(expiration time.Duration) func(o *auth.TokenOptions)
```
WithTokenExpiration returns a function that can be passed to GenerateToken
to set the expiration time of the generated token.



## Types
### Type Client
```go
type Client interface {
	GetVpcEndpointServiceName(ctx context.Context, params *dsql.GetVpcEndpointServiceNameInput, optFns ...func(*dsql.Options)) (*dsql.GetVpcEndpointServiceNameOutput, error)
}
```
Client is a minimal interface for interacting with DSQL operations needed to
manage VPC endpoints.


### Type Cluster
```go
type Cluster struct {
	// contains filtered or unexported fields
}
```
Cluster represents a DSQL cluster and provides methods to retrieve
information about it.

### Functions

```go
func NewCluster(cfg aws.Config, id string, opts ...Option) (*Cluster, error)
```
NewCluster creates a new Cluster instance for the given cluster ID.



### Methods

```go
func (c *Cluster) GetPrivateLinkServiceName(ctx context.Context) (string, error)
```
GetPrivateLinkServiceName retrieves the VPC endpoint service name for the
cluster.




### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithDSQLClient(client Client) Option
```
WithDSQLClient returns an Option that allows specifying a custom DSQL client
implementation, which can be useful for testing or if you want to use a
pre-configured client.







