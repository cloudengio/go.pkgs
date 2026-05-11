// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc

import (
	"context"
	"fmt"

	"cloudeng.io/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DescribeEndpoints is like DescribeEndpoints but includes a filter to
// restrict results to endpoints in the VPC.
func (v *T) DescribeEndpoints(ctx context.Context, ids []string, filters ...types.Filter) ([]Endpoint, error) {
	optsOrFilters := make([]any, 0, len(filters)+2)
	optsOrFilters = append(optsOrFilters, WithClient(v.client))
	optsOrFilters = append(optsOrFilters, types.Filter{Name: aws.String("vpc-id"), Values: []string{v.id}})
	for _, f := range filters {
		optsOrFilters = append(optsOrFilters, f)
	}
	return DescribeEndpoints(ctx, ids, optsOrFilters...)
}

// DescribeEndpoints returns the VPC endpoints matching the given endpoint IDs and/or filters.
// optsOrFilters may be a mix of types.Filter and Option values (e.g. WithClient).
// ids narrows the results to specific endpoint IDs; pass nil to return all.
// Filters are ANDed together. The context must carry an aws.Config (see
// awsconfig.ContextWith) unless a client is supplied via WithClient.
func DescribeEndpoints(ctx context.Context, ids []string, optsOrFilters ...any) ([]Endpoint, error) {

	var filters []types.Filter
	var opts []Option
	for _, opt := range optsOrFilters {
		switch v := opt.(type) {
		case types.Filter:
			filters = append(filters, v)
		case *types.Filter:
			filters = append(filters, *v)
		case Option:
			opts = append(opts, v)
		default:
			return nil, fmt.Errorf("invalid option/filter type %T: expected either types.Filter or Option", opt)
		}
	}
	options, err := handleOptions(ctx, opts)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeVpcEndpointsInput{Filters: filters}
	if len(ids) > 0 {
		input.VpcEndpointIds = ids
	}

	var results []Endpoint
	paginator := ec2.NewDescribeVpcEndpointsPaginator(options.client, input)
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list endpoints: %w", err)
		}
		for _, ep := range out.VpcEndpoints {
			results = append(results, endpointFromAPI(ep))
		}
	}
	return results, nil
}

// DeleteEndpoint deletes the VPC endpoint with the given ID.
func (v *T) DeleteEndpoint(ctx context.Context, endpointIDs ...string) error {
	if len(endpointIDs) == 0 {
		return fmt.Errorf("vpc %s: no endpoint IDs provided for deletion", v.id)
	}
	out, err := v.client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: endpointIDs,
	})
	if err != nil {
		return fmt.Errorf("vpc %s: delete endpoint %v: %w", v.id, endpointIDs, err)
	}
	// The API returns per-item errors for endpoints it could not delete.
	var errs errors.M
	for _, u := range out.Unsuccessful {
		id := aws.ToString(u.ResourceId)
		code, msg := "", ""
		if u.Error != nil {
			code = aws.ToString(u.Error.Code)
			msg = aws.ToString(u.Error.Message)
		}
		errs.Append(fmt.Errorf("vpc %s: delete endpoint %s: %s: %s", v.id, id, code, msg))
	}
	if err := errs.Err(); err != nil {
		return err
	}
	return nil
}

// Params returns an ec2.CreateVpcEndpointInput populated from the endpoint's fields.
// Read-only fields returned by the AWS API (ID, VPCID, OwnerID, State,
// NetworkInterfaceIDs, DNS entries, creation time, etc.) are not included —
// only the fields that are meaningful inputs to CreateEndpoint.
// VpcId is not set; it is supplied by CreateEndpoint from the VPC.
// ClientToken, DryRun, and SubnetConfigurations are left at their zero values
// and must be set by the caller if needed.
func (e Endpoint) Params() ec2.CreateVpcEndpointInput {
	input := ec2.CreateVpcEndpointInput{
		ServiceName:       aws.String(e.ServiceName),
		VpcEndpointType:   e.Type,
		SubnetIds:         e.SubnetIDs,
		SecurityGroupIds:  e.SecurityGroupIDs,
		RouteTableIds:     e.RouteTableIDs,
		PrivateDnsEnabled: aws.Bool(e.PrivateDNSEnabled),
		IpAddressType:     e.IpAddressType,
	}
	if e.PolicyDocument != "" {
		input.PolicyDocument = aws.String(e.PolicyDocument)
	}
	if e.ResourceConfigurationARN != "" {
		input.ResourceConfigurationArn = aws.String(e.ResourceConfigurationARN)
	}
	if e.ServiceNetworkARN != "" {
		input.ServiceNetworkArn = aws.String(e.ServiceNetworkARN)
	}
	if e.ServiceRegion != "" {
		input.ServiceRegion = aws.String(e.ServiceRegion)
	}
	if e.DnsOptions != nil {
		input.DnsOptions = &types.DnsOptionsSpecification{
			DnsRecordIpType: e.DnsOptions.DNSRecordIPType,
		}
		if e.DnsOptions.PrivateDNSOnlyForInboundResolverEndpoint {
			input.DnsOptions.PrivateDnsOnlyForInboundResolverEndpoint = aws.Bool(e.DnsOptions.PrivateDNSOnlyForInboundResolverEndpoint)
		}
		if len(e.DnsOptions.PrivateDNSPreference) > 0 {
			input.DnsOptions.PrivateDnsPreference = aws.String(e.DnsOptions.PrivateDNSPreference)
		}
	}
	for _, tg := range e.Tags {
		if input.TagSpecifications == nil {
			input.TagSpecifications = []types.TagSpecification{{ResourceType: types.ResourceTypeVpcEndpoint}}
		}
		input.TagSpecifications[0].Tags = append(input.TagSpecifications[0].Tags, types.Tag{Key: aws.String(tg.Name), Value: aws.String(tg.Value)})
	}
	return input
}

// CreateEndpoint creates a VPC endpoint in the VPC. It returns the new endpoint ID.
// input.VpcId is overwritten with the VPC's ID.
func (v *T) CreateEndpoint(ctx context.Context, ep Endpoint) (string, error) {
	input := ep.Params()
	if err := validateEndpointInput(input); err != nil {
		return "", fmt.Errorf("vpc %s: invalid endpoint params: %w", v.id, err)
	}
	input.VpcId = aws.String(v.id)
	out, err := v.client.CreateVpcEndpoint(ctx, &input)
	if err != nil {
		return "", fmt.Errorf("vpc %s: create endpoint for %s: %w", v.id, aws.ToString(input.ServiceName), err)
	}
	return aws.ToString(out.VpcEndpoint.VpcEndpointId), nil
}

func validateEndpointInput(input ec2.CreateVpcEndpointInput) error {
	if aws.ToString(input.ServiceName) == "" {
		return fmt.Errorf("ServiceName is required")
	}
	switch input.VpcEndpointType {
	case types.VpcEndpointTypeGateway:
		if len(input.RouteTableIds) == 0 {
			return fmt.Errorf("RouteTableIDs is required for gateway endpoints")
		}
	case types.VpcEndpointTypeInterface:
		if len(input.SubnetIds) == 0 {
			return fmt.Errorf("SubnetIDs is required for interface endpoints")
		}
		if len(input.SecurityGroupIds) == 0 {
			return fmt.Errorf("SecurityGroupIDs is required for interface endpoints")
		}
	case types.VpcEndpointTypeGatewayLoadBalancer,
		types.VpcEndpointTypeResource,
		types.VpcEndpointTypeServiceNetwork:
		// accepted as-is; caller is responsible for any type-specific fields
	default:
		return fmt.Errorf("unsupported endpoint type %q", input.VpcEndpointType)
	}
	return nil
}
