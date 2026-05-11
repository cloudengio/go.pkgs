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


### Type DNSEntry
```go
type DNSEntry struct {
	DNSName      string `yaml:"dns_name"`
	HostedZoneID string `yaml:"hosted_zone_id"`
}
```
DNSEntry holds the DNS name and hosted zone for a VPC endpoint DNS record.


### Type DNSOptions
```go
type DNSOptions struct {
	DNSRecordIPType                          types.DnsRecordIpType `yaml:"dns_record_ip_type"`
	PrivateDNSOnlyForInboundResolverEndpoint bool                  `yaml:"private_dns_only_for_inbound_resolver_endpoint"`
	PrivateDNSPreference                     string                `yaml:"private_dns_preference"`
	PrivateDNSSpecifiedDomains               []string              `yaml:"private_dns_specified_domains"`
}
```
DNSOptions holds the DNS configuration for a VPC endpoint.


### Type Endpoint
```go
type Endpoint struct {
	// Core identity
	ID          string                `yaml:"id"`
	VPCID       string                `yaml:"vpc_id"`
	OwnerID     string                `yaml:"owner_id"`
	ServiceName string                `yaml:"service_name"`
	Type        types.VpcEndpointType `yaml:"type"`
	State       types.State           `yaml:"state"`

	// Networking
	SubnetIDs           []string            `yaml:"subnet_ids"`
	SecurityGroupIDs    []string            `yaml:"security_group_ids"`
	RouteTableIDs       []string            `yaml:"route_table_ids"`
	NetworkInterfaceIDs []string            `yaml:"network_interface_ids"`
	IPAddressType       types.IpAddressType `yaml:"ip_address_type"`
	IPv4Prefixes        []SubnetIPPrefixes  `yaml:"ipv4_prefixes"`
	IPv6Prefixes        []SubnetIPPrefixes  `yaml:"ipv6_prefixes"`

	// DNS
	DNSEntries        []DNSEntry  `yaml:"dns_entries"`
	DNSOptions        *DNSOptions `yaml:"dns_options"`
	PrivateDNSEnabled bool        `yaml:"private_dns_enabled"`

	// Policy and routing
	PolicyDocument string `yaml:"policy_document"`

	// Service topology
	ServiceNetworkARN        string `yaml:"service_network_arn"`
	ServiceRegion            string `yaml:"service_region"`
	ResourceConfigurationARN string `yaml:"resource_configuration_arn"`

	// Management
	CreatedAt        *time.Time     `yaml:"created_at"`
	RequesterManaged bool           `yaml:"requester_managed"`
	FailureReason    string         `yaml:"failure_reason"`
	LastError        *EndpointError `yaml:"last_error"`

	Tags []Tag `yaml:"tags"`
}
```
Endpoint describes an existing VPC endpoint, mirroring all fields of
types.VpcEndpoint with Go-idiomatic naming and value (not pointer) scalars.

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



### Methods

```go
func (e Endpoint) Params() ec2.CreateVpcEndpointInput
```
Params returns an ec2.CreateVpcEndpointInput populated from the endpoint's
fields. Read-only fields returned by the AWS API (ID, VPCID, OwnerID, State,
NetworkInterfaceIDs, DNS entries, creation time, etc.) are not included —
only the fields that are meaningful inputs to CreateEndpoint. VpcId is not
set; it is supplied by CreateEndpoint from the VPC. ClientToken, DryRun,
and SubnetConfigurations are left at their zero values and must be set by
the caller if needed.




### Type EndpointError
```go
type EndpointError struct {
	Code    string `yaml:"code"`
	Message string `yaml:"message"`
}
```
EndpointError holds the last error recorded for a VPC endpoint.


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


```go
func WithConfig(cfg aws.Config) Option
```
WithConfig allows callers to specify an aws.Config directly, which will be
used to create a default client. If not provided, the aws.Config will be
retrieved from the context (see awsconfig.ContextWith) to create the default
client. If a config is provided, the context does not need to carry an
aws.Config.




### Type SecurityGroupInfo
```go
type SecurityGroupInfo struct {
	ID   string
	Name string
}
```
SecurityGroupInfo holds the essential details of a security group.


### Type SubnetIPPrefixes
```go
type SubnetIPPrefixes struct {
	SubnetID   string   `yaml:"subnet_id"`
	IPPrefixes []string `yaml:"ip_prefixes"`
}
```
SubnetIPPrefixes holds the IP prefix allocation for a subnet within an
endpoint.


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
func NewVPC(ctx context.Context, id string, opts ...Option) (*T, error)
```
NewVPC creates a new T instance for the given VPC ID using the provided
AWS config and options. It will only fail if WithClient is not provided,
WithConfig is not provided, and the context does not carry an aws.Config.



### Methods

```go
func (v *T) CreateEndpoint(ctx context.Context, ep Endpoint) (string, error)
```
CreateEndpoint creates a VPC endpoint in the VPC. It returns the new
endpoint ID. input.VpcId is overwritten with the VPC's ID.


```go
func (v *T) DeleteEndpoint(ctx context.Context, endpointIDs ...string) error
```
DeleteEndpoint deletes the VPC endpoint with the given ID.


```go
func (v *T) Describe(ctx context.Context) error
```
Describe queries the AWS EC2 API to gather all information about the VPC
required to create and delete endpoints: subnets, security groups, route
tables, and any existing endpoints. The result is stored in T.Config.


```go
func (v *T) DescribeEndpoints(ctx context.Context, ids []string, filters ...types.Filter) ([]Endpoint, error)
```
DescribeEndpoints is like DescribeEndpoints but includes a filter to
restrict results to endpoints in the VPC.




### Type Tag
```go
type Tag struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
```

### Methods

```go
func (t Tag) String() string
```







