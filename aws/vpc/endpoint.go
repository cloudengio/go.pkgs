// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DeleteEndpoint deletes the VPC endpoint with the given ID.
func (v *T) DeleteEndpoint(ctx context.Context, endpointIDs ...string) error {
	out, err := v.client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: endpointIDs,
	})
	if err != nil {
		return fmt.Errorf("vpc %s: delete endpoint %v: %w", v.id, endpointIDs, err)
	}
	// The API returns per-item errors for endpoints it could not delete.
	if len(out.Unsuccessful) > 0 {
		u := out.Unsuccessful[0]
		code, msg := "", ""
		if u.Error != nil {
			if u.Error.Code != nil {
				code = *u.Error.Code
			}
			if u.Error.Message != nil {
				msg = *u.Error.Message
			}
		}
		return fmt.Errorf("vpc %s: delete endpoints %v: %s: %s", v.id, endpointIDs, code, msg)
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
