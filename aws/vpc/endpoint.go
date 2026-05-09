// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc

import (
	"context"
	"fmt"

	"cloudeng.io/aws/awsconfig"
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
	var options options
	for _, opt := range opts {
		opt(&options)
	}
	if options.client == nil {
		cfg, ok := awsconfig.FromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("aws config not found in context")
		}
		options.client = ec2.NewFromConfig(cfg)
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

// EndpointParams holds the parameters required to create a VPC endpoint.
// SubnetIDs, SecurityGroupIDs and PrivateDNS apply to interface endpoints.
// RouteTableIDs applies to gateway endpoints.
type EndpointParams struct {
	ServiceName      string                `yaml:"serviceName"`
	Type             types.VpcEndpointType `yaml:"type"`
	SubnetIDs        []string              `yaml:"subnetIDs"`
	SecurityGroupIDs []string              `yaml:"securityGroupIDs"`
	RouteTableIDs    []string              `yaml:"routeTableIDs"`
	PrivateDNS       bool                  `yaml:"privateDNS"`
	Tags             []types.Tag
}

// CreateEndpoint creates a VPC endpoint in the VPC. It returns the new endpoint ID.
func (v *T) CreateEndpoint(ctx context.Context, params EndpointParams) (string, error) {
	if err := params.validate(); err != nil {
		return "", fmt.Errorf("vpc %s: invalid endpoint params: %w", v.id, err)
	}
	input := &ec2.CreateVpcEndpointInput{
		VpcId:           aws.String(v.id),
		ServiceName:     aws.String(params.ServiceName),
		VpcEndpointType: params.Type,
	}
	switch params.Type {
	case types.VpcEndpointTypeInterface:
		input.SubnetIds = params.SubnetIDs
		input.SecurityGroupIds = params.SecurityGroupIDs
		input.PrivateDnsEnabled = aws.Bool(params.PrivateDNS)
	case types.VpcEndpointTypeGateway:
		input.RouteTableIds = params.RouteTableIDs
	}
	if len(params.Tags) > 0 {
		input.TagSpecifications = []types.TagSpecification{{
			ResourceType: types.ResourceTypeVpcEndpoint,
			Tags:         params.Tags,
		}}
	}
	out, err := v.client.CreateVpcEndpoint(ctx, input)
	if err != nil {
		return "", fmt.Errorf("vpc %s: create endpoint for %s: %w", v.id, params.ServiceName, err)
	}
	return aws.ToString(out.VpcEndpoint.VpcEndpointId), nil
}

func (p EndpointParams) validate() error {
	if p.ServiceName == "" {
		return fmt.Errorf("ServiceName is required")
	}
	switch p.Type {
	case types.VpcEndpointTypeGateway:
		if len(p.RouteTableIDs) == 0 {
			return fmt.Errorf("RouteTableIDs is required for gateway endpoints")
		}
	case types.VpcEndpointTypeInterface:
		if len(p.SubnetIDs) == 0 {
			return fmt.Errorf("SubnetIDs is required for interface endpoints")
		}
		if len(p.SecurityGroupIDs) == 0 {
			return fmt.Errorf("SecurityGroupIDs is required for interface endpoints")
		}
	case types.VpcEndpointTypeGatewayLoadBalancer,
		types.VpcEndpointTypeResource,
		types.VpcEndpointTypeServiceNetwork:
		// accepted as-is; caller is responsible for any type-specific fields
	default:
		return fmt.Errorf("unsupported endpoint type %q", p.Type)
	}
	return nil
}
