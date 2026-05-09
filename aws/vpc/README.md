# Package [cloudeng.io/aws/vpc](https://pkg.go.dev/cloudeng.io/aws/vpc?tab=doc)

```go
import cloudeng.io/aws/vpc
```

Package vpc provides utilities for working with AWS VPCs.

## Types
### Type Client
```go
type Client interface {
	CreateVpcEndpoint(ctx context.Context, params *ec2.CreateVpcEndpointInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcEndpointOutput, error)
	DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
}
```
Client defines the methods required to manage VPC endpoints.


### Type Config
```go
type Config struct {
	VPCID          string              `yaml:"vpc_id"`
	Subnets        []SubnetInfo        `yaml:"subnets"`
	SecurityGroups []SecurityGroupInfo `yaml:"security_groups"`
	RouteTableIDs  []string            `yaml:"route_table_ids"`
	Endpoints      []Endpoint          `yaml:"endpoints"`
}
```
Config holds all VPC information required to create and delete endpoints.


### Type Endpoint
```go
type Endpoint struct {
	ID               string                `yaml:"id"`
	ServiceName      string                `yaml:"service_name"`
	Type             types.VpcEndpointType `yaml:"type"`
	State            types.State           `yaml:"state"`
	SubnetIDs        []string              `yaml:"subnet_ids"`
	SecurityGroupIDs []string              `yaml:"security_group_ids"`
	RouteTableIDs    []string              `yaml:"route_table_ids"`
}
```
Endpoint describes an existing VPC endpoint.

### Functions

```go
func DescribeEndpoints(ctx context.Context, ids []string, optsOrFilters ...any) ([]Endpoint, error)
```
DescribeEndpoints returns the VPC endpoints matching the given endpoint
IDs and/or filters. optsOrFilters may be a mix of types.Filter and Option
values (e.g. WithClient). ids narrows the results to specific endpoint IDs;
pass nil to return all. Filters are ANDed together. The context must carry
an aws.Config (see awsconfig.ContextWith) unless a client is supplied via
WithClient.




### Type EndpointParams
```go
type EndpointParams struct {
	ServiceName      string                `yaml:"serviceName"`
	Type             types.VpcEndpointType `yaml:"type"`
	SubnetIDs        []string              `yaml:"subnetIDs"`
	SecurityGroupIDs []string              `yaml:"securityGroupIDs"`
	RouteTableIDs    []string              `yaml:"routeTableIDs"`
	PrivateDNS       bool                  `yaml:"privateDNS"`
	Tags             []types.Tag
}
```
EndpointParams holds the parameters required to create a VPC endpoint.
SubnetIDs, SecurityGroupIDs and PrivateDNS apply to interface endpoints.
RouteTableIDs applies to gateway endpoints.


### Type Option
```go
type Option func(*options)
```
Option represents an option to multiple functions in this package.

### Functions

```go
func WithClient(client Client) Option
```
WithClient allows callers to specify a custom Client implementation
(e.g. for testing). If not provided, a default client will be
automatically created from the aws.Config stored in the context (see
awsconfig.ContextWith). If a client is provided, the context does not need
to carry an aws.Config.




### Type SecurityGroupInfo
```go
type SecurityGroupInfo struct {
	ID   string
	Name string
}
```
SecurityGroupInfo holds the essential details of a security group.


### Type SubnetInfo
```go
type SubnetInfo struct {
	ID               string
	AvailabilityZone string
	CIDRBlock        string
}
```
SubnetInfo holds the essential details of a VPC subnet.


### Type T
```go
type T struct {
	Config *Config
	// contains filtered or unexported fields
}
```
T represents a VPC whose configuration can be read via ReadConfig.

### Functions

```go
func NewVPC(cfg aws.Config, id string, opts ...Option) *T
```
NewVPC creates a new T instance for the given VPC ID using the provided AWS
config and options.



### Methods

```go
func (v *T) CreateEndpoint(ctx context.Context, params EndpointParams) (string, error)
```
CreateEndpoint creates a VPC endpoint in the VPC. It returns the new
endpoint ID.


```go
func (v *T) DeleteEndpoint(ctx context.Context, endpointIDs ...string) error
```
DeleteEndpoint deletes the VPC endpoint with the given ID.


```go
func (v *T) DescribeEndpoints(ctx context.Context, ids []string, filters ...types.Filter) ([]Endpoint, error)
```
DescribeEndpoints is like DescribeEndpoints but includes a filter to
restrict results to endpoints in the VPC.


```go
func (v *T) ReadConfig(ctx context.Context) error
```
ReadConfig queries the AWS EC2 API to gather all information about the
VPC required to create and delete endpoints: subnets, security groups,
route tables, and any existing endpoints. The result is stored in T.Config.







