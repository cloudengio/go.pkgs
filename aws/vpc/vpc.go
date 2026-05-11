// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package vpc provides utilities for working with AWS VPCs.
package vpc

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/aws/awsconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Client defines the methods required to manage VPC endpoints.
type Client interface {
	CreateVpcEndpoint(ctx context.Context, params *ec2.CreateVpcEndpointInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcEndpointOutput, error)
	DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
}

// Option represents an option to multiple functions in this package.
type Option func(*options)

type options struct {
	client    Client
	config    aws.Config
	hasConfig bool
}

// WithClient allows callers to specify a custom Client implementation (e.g.
// for testing). If not provided, a default client will be automatically
// created from the aws.Config stored in the context (see awsconfig.ContextWith).
// If a client is provided, the context does not need to carry an aws.Config.
func WithClient(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithConfig allows callers to specify an aws.Config directly, which will be
// used to create a default client. If not provided, the aws.Config will be
// retrieved from the context (see awsconfig.ContextWith) to create the default
// client. If a config is provided, the context does not need to carry an
// aws.Config.
func WithConfig(cfg aws.Config) Option {
	return func(o *options) {
		o.config = cfg
		o.hasConfig = true
	}
}

// T represents a VPC whose configuration can be read via Describe.
type T struct {
	id     string
	client Client
	Config *Config
}

// SubnetInfo holds the essential details of a VPC subnet.
type SubnetInfo struct {
	ID               string
	AvailabilityZone string
	CIDRBlock        string
}

// SecurityGroupInfo holds the essential details of a security group.
type SecurityGroupInfo struct {
	ID   string
	Name string
}

// DNSEntry holds the DNS name and hosted zone for a VPC endpoint DNS record.
type DNSEntry struct {
	DNSName      string `yaml:"dns_name"`
	HostedZoneID string `yaml:"hosted_zone_id"`
}

// DNSOptions holds the DNS configuration for a VPC endpoint.
type DNSOptions struct {
	DNSRecordIPType                          types.DnsRecordIpType `yaml:"dns_record_ip_type"`
	PrivateDNSOnlyForInboundResolverEndpoint bool                  `yaml:"private_dns_only_for_inbound_resolver_endpoint"`
	PrivateDNSPreference                     string                `yaml:"private_dns_preference"`
	PrivateDNSSpecifiedDomains               []string              `yaml:"private_dns_specified_domains"`
}

// SubnetIPPrefixes holds the IP prefix allocation for a subnet within an endpoint.
type SubnetIPPrefixes struct {
	SubnetID   string   `yaml:"subnet_id"`
	IPPrefixes []string `yaml:"ip_prefixes"`
}

// EndpointError holds the last error recorded for a VPC endpoint.
type EndpointError struct {
	Code    string `yaml:"code"`
	Message string `yaml:"message"`
}

// Endpoint describes an existing VPC endpoint, mirroring all fields of
// types.VpcEndpoint with Go-idiomatic naming and value (not pointer) scalars.
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

type Tag struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (t Tag) String() string {
	return t.Name + "=" + t.Value
}

// Config holds all VPC information required to create and delete endpoints.
type Config struct {
	VPCID          string              `yaml:"vpc_id"`
	Subnets        []SubnetInfo        `yaml:"subnets"`
	SecurityGroups []SecurityGroupInfo `yaml:"security_groups"`
	RouteTableIDs  []string            `yaml:"route_table_ids"`
	Endpoints      []Endpoint          `yaml:"endpoints"`
}

func handleOptions(ctx context.Context, opts []Option) (options, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.client != nil {
		return o, nil
	}
	if !o.hasConfig {
		cfg, ok := awsconfig.FromContext(ctx)
		if !ok {
			return o, fmt.Errorf("aws config not found in context")
		}
		o.config = cfg
	}
	o.client = ec2.NewFromConfig(o.config)
	return o, nil
}

// NewVPC creates a new T instance for the given VPC ID using the provided
// AWS config and options. It will only fail if both WithClient and WithConfig
// are not provided and the context does not carry an aws.Config.
func NewVPC(ctx context.Context, id string, opts ...Option) (*T, error) {
	options, err := handleOptions(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &T{
		id:     id,
		client: options.client,
	}, nil
}

// Describe queries the AWS EC2 API to gather all information about the VPC
// required to create and delete endpoints: subnets, security groups, route
// tables, and any existing endpoints. The result is stored in T.Config.
func (v *T) Describe(ctx context.Context) error {
	cfg := &Config{VPCID: v.id}
	vpcFilter := []types.Filter{{Name: aws.String("vpc-id"), Values: []string{v.id}}}

	subnets, err := describeSubnets(ctx, v.client, vpcFilter)
	if err != nil {
		return fmt.Errorf("vpc %s: describe subnets: %w", v.id, err)
	}
	cfg.Subnets = subnets

	sgs, err := describeSecurityGroups(ctx, v.client, vpcFilter)
	if err != nil {
		return fmt.Errorf("vpc %s: describe security groups: %w", v.id, err)
	}
	cfg.SecurityGroups = sgs

	rtIDs, err := describeRouteTables(ctx, v.client, vpcFilter)
	if err != nil {
		return fmt.Errorf("vpc %s: describe route tables: %w", v.id, err)
	}
	cfg.RouteTableIDs = rtIDs

	endpoints, err := describeVpcEndpoints(ctx, v.client, vpcFilter)
	if err != nil {
		return fmt.Errorf("vpc %s: describe endpoints: %w", v.id, err)
	}
	cfg.Endpoints = endpoints

	v.Config = cfg
	return nil
}

func describeSubnets(ctx context.Context, client Client, filters []types.Filter) ([]SubnetInfo, error) {
	var results []SubnetInfo
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{Filters: filters})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range out.Subnets {
			results = append(results, SubnetInfo{
				ID:               aws.ToString(s.SubnetId),
				AvailabilityZone: aws.ToString(s.AvailabilityZone),
				CIDRBlock:        aws.ToString(s.CidrBlock),
			})
		}
	}
	return results, nil
}

func describeSecurityGroups(ctx context.Context, client Client, filters []types.Filter) ([]SecurityGroupInfo, error) {
	var results []SecurityGroupInfo
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{Filters: filters})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sg := range out.SecurityGroups {
			results = append(results, SecurityGroupInfo{
				ID:   aws.ToString(sg.GroupId),
				Name: aws.ToString(sg.GroupName),
			})
		}
	}
	return results, nil
}

func describeRouteTables(ctx context.Context, client Client, filters []types.Filter) ([]string, error) {
	var results []string
	paginator := ec2.NewDescribeRouteTablesPaginator(client, &ec2.DescribeRouteTablesInput{Filters: filters})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rt := range out.RouteTables {
			results = append(results, aws.ToString(rt.RouteTableId))
		}
	}
	return results, nil
}

func describeVpcEndpoints(ctx context.Context, client Client, filters []types.Filter) ([]Endpoint, error) {
	var results []Endpoint
	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{Filters: filters})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ep := range out.VpcEndpoints {
			results = append(results, endpointFromAPI(ep))
		}
	}
	return results, nil
}

func endpointFromAPI(ep types.VpcEndpoint) Endpoint {
	e := Endpoint{
		ID:                       aws.ToString(ep.VpcEndpointId),
		VPCID:                    aws.ToString(ep.VpcId),
		OwnerID:                  aws.ToString(ep.OwnerId),
		ServiceName:              aws.ToString(ep.ServiceName),
		Type:                     ep.VpcEndpointType,
		State:                    ep.State,
		SubnetIDs:                ep.SubnetIds,
		RouteTableIDs:            ep.RouteTableIds,
		NetworkInterfaceIDs:      ep.NetworkInterfaceIds,
		IPAddressType:            ep.IpAddressType,
		PrivateDNSEnabled:        aws.ToBool(ep.PrivateDnsEnabled),
		PolicyDocument:           aws.ToString(ep.PolicyDocument), // PolicyDocument is a string, not a pointer
		ServiceNetworkARN:        aws.ToString(ep.ServiceNetworkArn),
		ServiceRegion:            aws.ToString(ep.ServiceRegion),
		ResourceConfigurationARN: aws.ToString(ep.ResourceConfigurationArn),
		CreatedAt:                ep.CreationTimestamp,
		RequesterManaged:         aws.ToBool(ep.RequesterManaged),
		FailureReason:            aws.ToString(ep.FailureReason),
	}
	for _, p := range ep.Ipv4Prefixes {
		e.IPv4Prefixes = append(e.IPv4Prefixes, SubnetIPPrefixes{SubnetID: aws.ToString(p.SubnetId), IPPrefixes: p.IpPrefixes})
	}
	for _, p := range ep.Ipv6Prefixes {
		e.IPv6Prefixes = append(e.IPv6Prefixes, SubnetIPPrefixes{SubnetID: aws.ToString(p.SubnetId), IPPrefixes: p.IpPrefixes})
	}
	for _, d := range ep.DnsEntries {
		e.DNSEntries = append(e.DNSEntries, DNSEntry{DNSName: aws.ToString(d.DnsName), HostedZoneID: aws.ToString(d.HostedZoneId)})
	}
	if o := ep.DnsOptions; o != nil {
		e.DNSOptions = &DNSOptions{
			DNSRecordIPType:                          o.DnsRecordIpType,
			PrivateDNSOnlyForInboundResolverEndpoint: aws.ToBool(o.PrivateDnsOnlyForInboundResolverEndpoint),
			PrivateDNSPreference:                     aws.ToString(o.PrivateDnsPreference),
			PrivateDNSSpecifiedDomains:               o.PrivateDnsSpecifiedDomains,
		}
	}
	if le := ep.LastError; le != nil {
		e.LastError = &EndpointError{Code: aws.ToString(le.Code), Message: aws.ToString(le.Message)}
	}
	for _, sg := range ep.Groups {
		e.SecurityGroupIDs = append(e.SecurityGroupIDs, aws.ToString(sg.GroupId))
	}
	for _, tg := range ep.Tags {
		e.Tags = append(e.Tags, Tag{Name: aws.ToString(tg.Key), Value: aws.ToString(tg.Value)})
	}
	return e
}
