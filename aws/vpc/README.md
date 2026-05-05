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
	VPCID          string
	Subnets        []SubnetInfo
	SecurityGroups []SecurityGroupInfo
	RouteTableIDs  []string
	Endpoints      []Endpoint
}
```
Config holds all VPC information required to create and delete endpoints.


### Type Endpoint
```go
type Endpoint struct {
	ID               string
	ServiceName      string
	Type             types.VpcEndpointType
	State            types.State
	SubnetIDs        []string
	SecurityGroupIDs []string
	RouteTableIDs    []string
}
```
Endpoint describes an existing VPC endpoint.


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
func NewVPC(cfg aws.Config, id string) *T
```
NewVPC creates a new T instance for the given VPC ID.


```go
func NewVPCWithClient(id string, client Client) *T
```
NewVPCWithClient creates a T using an already-configured Client. Intended
for tests that inject a localstack-pointed client.



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
func (v *T) ReadConfig(ctx context.Context) error
```
ReadConfig queries the AWS EC2 API to gather all information about the
VPC required to create and delete endpoints: subnets, security groups,
route tables, and any existing endpoints. The result is stored in T.Config.







