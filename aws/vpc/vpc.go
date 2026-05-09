// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package vpc provides utilities for working with AWS VPCs.
package vpc

import (
	"context"
	"fmt"

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
	client Client
}

// WithClient allows callers to specify a custom Client implementation (e.g.
// for testing). If not provided, a default client will be automatically
// created from the aws.Config stored in the context (see awsconfig.ContextWith).
// If a client is provided, the context does not need to carry an aws.Config.
func WithClient(client Client) func(*options) {
	return func(o *options) {
		o.client = client
	}
}

// T represents a VPC whose configuration can be read via ReadConfig.
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

// Endpoint describes an existing VPC endpoint.
type Endpoint struct {
	ID               string                `yaml:"id"`
	ServiceName      string                `yaml:"service_name"`
	Type             types.VpcEndpointType `yaml:"type"`
	State            types.State           `yaml:"state"`
	SubnetIDs        []string              `yaml:"subnet_ids"`
	SecurityGroupIDs []string              `yaml:"security_group_ids"`
	RouteTableIDs    []string              `yaml:"route_table_ids"`
}

// Config holds all VPC information required to create and delete endpoints.
type Config struct {
	VPCID          string              `yaml:"vpc_id"`
	Subnets        []SubnetInfo        `yaml:"subnets"`
	SecurityGroups []SecurityGroupInfo `yaml:"securityGroups"`
	RouteTableIDs  []string            `yaml:"routeTableIDs"`
	Endpoints      []Endpoint          `yaml:"endpoints"`
}

func handleOptions(cfg aws.Config, opts []Option) options {
	var options options
	for _, opt := range opts {
		opt(&options)
	}
	if options.client == nil {
		options.client = ec2.NewFromConfig(cfg)
	}
	return options
}

// NewVPC creates a new T instance for the given VPC ID using the provided
// AWS config and options.
func NewVPC(cfg aws.Config, id string, opts ...Option) *T {
	options := handleOptions(cfg, opts)
	return &T{
		id:     id,
		client: options.client,
	}
}

// ReadConfig queries the AWS EC2 API to gather all information about the VPC
// required to create and delete endpoints: subnets, security groups, route
// tables, and any existing endpoints. The result is stored in T.Config.
func (v *T) ReadConfig(ctx context.Context) error {
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
			e := Endpoint{
				ID:            aws.ToString(ep.VpcEndpointId),
				ServiceName:   aws.ToString(ep.ServiceName),
				Type:          ep.VpcEndpointType,
				State:         ep.State,
				SubnetIDs:     ep.SubnetIds,
				RouteTableIDs: ep.RouteTableIds,
			}
			for _, sg := range ep.Groups {
				e.SecurityGroupIDs = append(e.SecurityGroupIDs, aws.ToString(sg.GroupId))
			}
			results = append(results, e)
		}
	}
	return results, nil
}
