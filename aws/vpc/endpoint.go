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
