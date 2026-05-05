// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc_test

import (
	"context"
	"slices"
	"testing"

	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/vpc"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var awsService *awstestutil.AWS

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
		client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{RouteTableId: aws.String(rtID)})           //nolint:errcheck
		client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(sgID)})         //nolint:errcheck
		client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: aws.String(subnetID)})                  //nolint:errcheck
		client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: aws.String(vpcID)})                              //nolint:errcheck
	}
	return
}

func TestReadConfig(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	cfg := awstestutil.DefaultAWSConfig()
	client := awsService.EC2(cfg)

	vpcID, subnetID, sgID, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPCWithClient(vpcID, client)
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
	ctx := context.Background()
	cfg := awstestutil.DefaultAWSConfig()
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPCWithClient(vpcID, client)

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

// TestDeleteEndpointNonexistent verifies that DeleteEndpoint does not return an
// error for an endpoint ID that was never created. The AWS DeleteVpcEndpoints
// API silently ignores unknown IDs rather than returning an Unsuccessful entry.
func TestDeleteEndpointNonexistent(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	cfg := awstestutil.DefaultAWSConfig()
	client := awsService.EC2(cfg)

	vpcID, _, _, _, cleanup := setupVPC(t, client)
	defer cleanup()

	v := vpc.NewVPCWithClient(vpcID, client)
	if err := v.DeleteEndpoint(ctx, "vpce-000000000000abcd"); err != nil {
		t.Errorf("unexpected error deleting nonexistent endpoint: %v", err)
	}
}
