// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/vpc"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var awsService *awstestutil.AWS

func TestCreateEndpointValidation(t *testing.T) {
	v := vpc.NewVPC(aws.Config{}, "vpc-test")
	for _, tc := range []struct {
		params vpc.EndpointParams
		errMsg string
	}{
		{
			vpc.EndpointParams{},
			"ServiceName is required",
		},
		{
			vpc.EndpointParams{ServiceName: "svc"},
			`unsupported endpoint type ""`,
		},
		{
			vpc.EndpointParams{ServiceName: "svc", Type: types.VpcEndpointTypeGateway},
			"RouteTableIDs is required for gateway endpoints",
		},
		{
			vpc.EndpointParams{ServiceName: "svc", Type: types.VpcEndpointTypeInterface},
			"SubnetIDs is required for interface endpoints",
		},
		{
			vpc.EndpointParams{ServiceName: "svc", Type: types.VpcEndpointTypeInterface, SubnetIDs: []string{"s-1"}},
			"SecurityGroupIDs is required for interface endpoints",
		},
	} {
		_, err := v.CreateEndpoint(context.Background(), tc.params)
		if err == nil {
			t.Errorf("params %+v: expected error %q, got nil", tc.params, tc.errMsg)
			continue
		}
		if !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("params %+v: got error %q, want it to contain %q", tc.params, err.Error(), tc.errMsg)
		}
	}
}

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsService, awstestutil.WithEC2())
}

// setupVPC creates a VPC with one subnet, one security group, one route table,
// and returns their IDs along with a cleanup function.
func setupVPC(t *testing.T, client *ec2.Client) (vpcID, subnetID, sgID, rtID string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	vpcOut, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	})
	if err != nil {
		t.Fatalf("CreateVpc: %v", err)
	}
	vpcID = aws.ToString(vpcOut.Vpc.VpcId)

	subnetOut, err := client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:            aws.String(vpcID),
		CidrBlock:        aws.String("10.0.1.0/24"),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if err != nil {
		t.Fatalf("CreateSubnet: %v", err)
	}
	subnetID = aws.ToString(subnetOut.Subnet.SubnetId)

	sgOut, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		VpcId:       aws.String(vpcID),
		GroupName:   aws.String("test-sg"),
		Description: aws.String("test security group"),
	})
	if err != nil {
		t.Fatalf("CreateSecurityGroup: %v", err)
	}
	sgID = aws.ToString(sgOut.GroupId)

	rtOut, err := client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		t.Fatalf("CreateRouteTable: %v", err)
	}
	rtID = aws.ToString(rtOut.RouteTable.RouteTableId)

	cleanup = func() {
		ctx := context.Background()
		client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{RouteTableId: aws.String(rtID)})  //nolint:errcheck
		client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(sgID)}) //nolint:errcheck
		client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: aws.String(subnetID)})          //nolint:errcheck
		client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: aws.String(vpcID)})                      //nolint:errcheck
	}
	return
}

func TestReadConfig(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := context.Background()
	client := awsService.EC2(cfg)

	vpcID, subnetID, sgID, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPC(cfg, vpcID, vpc.WithClient(client))
	if err := v.ReadConfig(ctx); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	cfg2 := v.Config
	if cfg2 == nil {
		t.Fatal("Config is nil after ReadConfig")
	}
	if cfg2.VPCID != vpcID {
		t.Errorf("VPCID: got %q, want %q", cfg2.VPCID, vpcID)
	}

	subnetIDs := make([]string, len(cfg2.Subnets))
	for i, s := range cfg2.Subnets {
		subnetIDs[i] = s.ID
	}
	if !slices.Contains(subnetIDs, subnetID) {
		t.Errorf("subnets %v does not contain %q", subnetIDs, subnetID)
	}

	sgIDs := make([]string, len(cfg2.SecurityGroups))
	for i, sg := range cfg2.SecurityGroups {
		sgIDs[i] = sg.ID
	}
	if !slices.Contains(sgIDs, sgID) {
		t.Errorf("security groups %v does not contain %q", sgIDs, sgID)
	}

	if !slices.Contains(cfg2.RouteTableIDs, rtID) {
		t.Errorf("route tables %v does not contain %q", cfg2.RouteTableIDs, rtID)
	}
}

func TestCreateAndDeleteEndpoint(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := context.Background()
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPC(cfg, vpcID, vpc.WithClient(client))

	endpointID, err := v.CreateEndpoint(ctx, vpc.EndpointParams{
		ServiceName:   "com.amazonaws.us-east-1.s3",
		Type:          types.VpcEndpointTypeGateway,
		RouteTableIDs: []string{rtID},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}
	if endpointID == "" {
		t.Fatal("CreateEndpoint returned empty ID")
	}

	// Verify the endpoint appears in ReadConfig.
	if err := v.ReadConfig(ctx); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	endpointIDs := make([]string, len(v.Config.Endpoints))
	for i, ep := range v.Config.Endpoints {
		endpointIDs[i] = ep.ID
	}
	if !slices.Contains(endpointIDs, endpointID) {
		t.Errorf("endpoints %v does not contain %q", endpointIDs, endpointID)
	}

	if err := v.DeleteEndpoint(ctx, endpointID); err != nil {
		t.Fatalf("DeleteEndpoint: %v", err)
	}
}

func TestDescribeEndpoints(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := awsconfig.ContextWith(context.Background(), cfg)
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPC(cfg, vpcID, vpc.WithClient(client))

	// No endpoints yet — list should be empty.
	eps, err := v.DescribeEndpoints(ctx, nil)
	if err != nil {
		t.Fatalf("DescribeEndpoints (empty): %v", err)
	}
	if len(eps) != 0 {
		t.Errorf("expected 0 endpoints before creation, got %d", len(eps))
	}

	// Create an endpoint.
	endpointID, err := v.CreateEndpoint(ctx, vpc.EndpointParams{
		ServiceName:   "com.amazonaws.us-east-1.s3",
		Type:          types.VpcEndpointTypeGateway,
		RouteTableIDs: []string{rtID},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}
	defer v.DeleteEndpoint(ctx, endpointID) //nolint:errcheck

	// List all — should contain the new endpoint.
	eps, err = v.DescribeEndpoints(ctx, nil)
	if err != nil {
		t.Fatalf("DescribeEndpoints (all): %v", err)
	}
	ids := make([]string, len(eps))
	for i, ep := range eps {
		ids[i] = ep.ID
	}
	if !slices.Contains(ids, endpointID) {
		t.Errorf("DescribeEndpoints: %v does not contain %q", ids, endpointID)
	}

	// Query by ID — should return exactly that endpoint.
	eps, err = v.DescribeEndpoints(ctx, []string{endpointID})
	if err != nil {
		t.Fatalf("DescribeEndpoints (by ID): %v", err)
	}
	if len(eps) != 1 || eps[0].ID != endpointID {
		t.Errorf("DescribeEndpoints by ID: got %v, want [%s]", eps, endpointID)
	}

	// Filter by service name — should match.
	eps, err = v.DescribeEndpoints(ctx, nil,
		types.Filter{Name: aws.String("service-name"), Values: []string{"com.amazonaws.us-east-1.s3"}},
	)
	if err != nil {
		t.Fatalf("DescribeEndpoints (service filter): %v", err)
	}
	if !slices.ContainsFunc(eps, func(ep vpc.Endpoint) bool { return ep.ID == endpointID }) {
		t.Errorf("DescribeEndpoints with service filter: %v does not contain %q", eps, endpointID)
	}

	// Filter by non-matching service name — should return nothing.
	eps, err = v.DescribeEndpoints(ctx, nil,
		types.Filter{Name: aws.String("service-name"), Values: []string{"com.amazonaws.us-east-1.nonexistent"}},
	)
	if err != nil {
		t.Fatalf("DescribeEndpoints (no-match filter): %v", err)
	}
	if len(eps) != 0 {
		t.Errorf("DescribeEndpoints with non-matching filter: expected 0, got %d", len(eps))
	}
}

// TestDescribeEndpointsPackageLevel exercises the package-level DescribeEndpoints
// function directly, bypassing the vpc-id scoping that the method adds.
func TestDescribeEndpointsPackageLevel(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := awsconfig.ContextWith(context.Background(), cfg)
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	// Create an endpoint inside our VPC.
	v := vpc.NewVPC(cfg, vpcID, vpc.WithClient(client))
	endpointID, err := v.CreateEndpoint(ctx, vpc.EndpointParams{
		ServiceName:   "com.amazonaws.us-east-1.s3",
		Type:          types.VpcEndpointTypeGateway,
		RouteTableIDs: []string{rtID},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}
	defer v.DeleteEndpoint(ctx, endpointID) //nolint:errcheck

	// Package-level call with explicit vpc-id filter should find the endpoint.
	eps, err := vpc.DescribeEndpoints(ctx, nil,
		vpc.WithClient(client),
		types.Filter{Name: aws.String("vpc-id"), Values: []string{vpcID}},
	)
	if err != nil {
		t.Fatalf("DescribeEndpoints (vpc-id filter): %v", err)
	}
	ids := make([]string, len(eps))
	for i, ep := range eps {
		ids[i] = ep.ID
	}
	if !slices.Contains(ids, endpointID) {
		t.Errorf("DescribeEndpoints: %v does not contain %q", ids, endpointID)
	}

	// Package-level call with a non-matching vpc-id should return nothing.
	eps, err = vpc.DescribeEndpoints(ctx, nil,
		vpc.WithClient(client),
		types.Filter{Name: aws.String("vpc-id"), Values: []string{"vpc-000000000000abcd"}},
	)
	if err != nil {
		t.Fatalf("DescribeEndpoints (non-matching vpc-id): %v", err)
	}
	if len(eps) != 0 {
		t.Errorf("expected 0 endpoints for non-matching vpc-id, got %d", len(eps))
	}

	// Package-level call with query by endpoint ID (no filters) should find it.
	eps, err = vpc.DescribeEndpoints(ctx, []string{endpointID}, vpc.WithClient(client))
	if err != nil {
		t.Fatalf("DescribeEndpoints (query by ID): %v", err)
	}
	if len(eps) != 1 || eps[0].ID != endpointID {
		t.Errorf("DescribeEndpoints by ID: got %v, want [%s]", eps, endpointID)
	}
}

// TestDeleteEndpointNonexistent verifies that DeleteEndpoint does not return an
// error for an endpoint ID that was never created. The AWS DeleteVpcEndpoints
// API silently ignores unknown IDs rather than returning an Unsuccessful entry.
func TestDeleteEndpointNonexistent(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := context.Background()
	client := awsService.EC2(cfg)

	vpcID, _, _, _, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPC(cfg, vpcID, vpc.WithClient(client))
	if err := v.DeleteEndpoint(ctx, "vpce-000000000000abcd"); err != nil {
		t.Errorf("unexpected error deleting nonexistent endpoint: %v", err)
	}
}
