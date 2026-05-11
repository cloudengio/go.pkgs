// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vpc_test

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/vpc"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"gopkg.in/yaml.v3"
)

var awsService *awstestutil.AWS

func TestCreateEndpointValidation(t *testing.T) {
	v, err := vpc.NewVPC(context.Background(), "vpc-test", vpc.WithConfig(aws.Config{}))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}
	for _, tc := range []struct {
		params vpc.Endpoint
		errMsg string
	}{
		{
			vpc.Endpoint{},
			"ServiceName is required",
		},
		{
			vpc.Endpoint{ServiceName: "svc"},
			`unsupported endpoint type ""`,
		},
		{
			vpc.Endpoint{ServiceName: "svc", Type: types.VpcEndpointTypeGateway},
			"RouteTableIDs is required for gateway endpoints",
		},
		{
			vpc.Endpoint{ServiceName: "svc", Type: types.VpcEndpointTypeInterface},
			"SubnetIDs is required for interface endpoints",
		},
		{
			vpc.Endpoint{ServiceName: "svc", Type: types.VpcEndpointTypeInterface, SubnetIDs: []string{"s-1"}},
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

func checkDnsOptions(t *testing.T, got *types.DnsOptionsSpecification, want *vpc.DNSOptions) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("DNSOptions: got %v, want nil", got)
		}
		return
	}
	if got == nil {
		t.Fatal("DnsOptions should not be nil")
		return
	}
	if got.DnsRecordIpType != want.DNSRecordIPType {
		t.Errorf("DnsOptions.DnsRecordIpType: got %v, want %v", got.DnsRecordIpType, want.DNSRecordIPType)
	}
	if aws.ToBool(got.PrivateDnsOnlyForInboundResolverEndpoint) != want.PrivateDNSOnlyForInboundResolverEndpoint {
		t.Errorf("DnsOptions.PrivateDnsOnlyForInboundResolverEndpoint: got %v, want %v",
			aws.ToBool(got.PrivateDnsOnlyForInboundResolverEndpoint), want.PrivateDNSOnlyForInboundResolverEndpoint)
	}
	if aws.ToString(got.PrivateDnsPreference) != want.PrivateDNSPreference {
		t.Errorf("DnsOptions.PrivateDnsPreference: got %q, want %q",
			aws.ToString(got.PrivateDnsPreference), want.PrivateDNSPreference)
	}
}

func checkTagSpecs(t *testing.T, specs []types.TagSpecification, want []vpc.Tag) {
	t.Helper()
	if len(want) == 0 {
		if specs != nil {
			t.Errorf("TagSpecifications should be nil for endpoint with no tags, got %v", specs)
		}
		return
	}
	if len(specs) == 0 {
		t.Fatal("TagSpecifications should not be empty")
		return
	}
	tags := specs[0].Tags
	if len(tags) != len(want) {
		t.Fatalf("Tags len: got %d, want %d", len(tags), len(want))
		return
	}
	for i, w := range want {
		if aws.ToString(tags[i].Key) != w.Name {
			t.Errorf("Tags[%d].Key: got %q, want %q", i, aws.ToString(tags[i].Key), w.Name)
		}
		if aws.ToString(tags[i].Value) != w.Value {
			t.Errorf("Tags[%d].Value: got %q, want %q", i, aws.ToString(tags[i].Value), w.Value)
		}
	}
}

func TestEndpointParams(t *testing.T) {
	ep := vpc.Endpoint{
		// Create-time fields — should appear in Params().
		ServiceName:              "com.amazonaws.us-east-1.s3",
		Type:                     types.VpcEndpointTypeInterface,
		SubnetIDs:                []string{"subnet-1", "subnet-2"},
		SecurityGroupIDs:         []string{"sg-1"},
		RouteTableIDs:            []string{"rtb-1"},
		PrivateDNSEnabled:        true,
		PolicyDocument:           `{"Version":"2012-10-17"}`,
		ResourceConfigurationARN: "arn:aws:vpc-lattice:us-east-1:123:resourceconfiguration/rcfg-x",
		ServiceNetworkARN:        "arn:aws:vpc-lattice:us-east-1:123:servicenetwork/sn-x",
		ServiceRegion:            "us-west-2",
		IPAddressType:            types.IpAddressTypeIpv4,
		DNSOptions: &vpc.DNSOptions{
			DNSRecordIPType:                          types.DnsRecordIpTypeIpv4,
			PrivateDNSOnlyForInboundResolverEndpoint: true,
			PrivateDNSPreference:                     "ALL_DOMAINS",
		},
		Tags: []vpc.Tag{
			{Name: "Env", Value: "test"},
			{Name: "Owner", Value: "team"},
		},
		// Read-only fields — must NOT affect Params().
		ID:                  "vpce-abc",
		VPCID:               "vpc-xyz",
		OwnerID:             "123456789",
		State:               types.StateAvailable,
		NetworkInterfaceIDs: []string{"eni-1"},
	}
	p := ep.Params()

	t.Run("StringFields", func(t *testing.T) {
		if aws.ToString(p.ServiceName) != ep.ServiceName {
			t.Errorf("ServiceName: got %q, want %q", aws.ToString(p.ServiceName), ep.ServiceName)
		}
		if aws.ToString(p.PolicyDocument) != ep.PolicyDocument {
			t.Errorf("PolicyDocument: got %q, want %q", aws.ToString(p.PolicyDocument), ep.PolicyDocument)
		}
		if aws.ToString(p.ResourceConfigurationArn) != ep.ResourceConfigurationARN {
			t.Errorf("ResourceConfigurationArn: got %q, want %q", aws.ToString(p.ResourceConfigurationArn), ep.ResourceConfigurationARN)
		}
		if aws.ToString(p.ServiceNetworkArn) != ep.ServiceNetworkARN {
			t.Errorf("ServiceNetworkArn: got %q, want %q", aws.ToString(p.ServiceNetworkArn), ep.ServiceNetworkARN)
		}
		if aws.ToString(p.ServiceRegion) != ep.ServiceRegion {
			t.Errorf("ServiceRegion: got %q, want %q", aws.ToString(p.ServiceRegion), ep.ServiceRegion)
		}
	})

	t.Run("CollectionFields", func(t *testing.T) {
		if p.VpcEndpointType != ep.Type {
			t.Errorf("VpcEndpointType: got %v, want %v", p.VpcEndpointType, ep.Type)
		}
		if !slices.Equal(p.SubnetIds, ep.SubnetIDs) {
			t.Errorf("SubnetIds: got %v, want %v", p.SubnetIds, ep.SubnetIDs)
		}
		if !slices.Equal(p.SecurityGroupIds, ep.SecurityGroupIDs) {
			t.Errorf("SecurityGroupIds: got %v, want %v", p.SecurityGroupIds, ep.SecurityGroupIDs)
		}
		if !slices.Equal(p.RouteTableIds, ep.RouteTableIDs) {
			t.Errorf("RouteTableIds: got %v, want %v", p.RouteTableIds, ep.RouteTableIDs)
		}
		if aws.ToBool(p.PrivateDnsEnabled) != ep.PrivateDNSEnabled {
			t.Errorf("PrivateDnsEnabled: got %v, want %v", aws.ToBool(p.PrivateDnsEnabled), ep.PrivateDNSEnabled)
		}
		if p.IpAddressType != ep.IPAddressType {
			t.Errorf("IPAddressType: got %v, want %v", p.IpAddressType, ep.IPAddressType)
		}
	})

	t.Run("DnsOptions", func(t *testing.T) {
		checkDnsOptions(t, p.DnsOptions, ep.DNSOptions)
	})

	t.Run("Tags", func(t *testing.T) {
		checkTagSpecs(t, p.TagSpecifications, ep.Tags)
	})

	t.Run("TransientFields", func(t *testing.T) {
		if aws.ToString(p.ClientToken) != "" {
			t.Errorf("ClientToken should be empty, got %q", aws.ToString(p.ClientToken))
		}
		if aws.ToBool(p.DryRun) {
			t.Error("DryRun should be false")
		}
		if p.SubnetConfigurations != nil {
			t.Errorf("SubnetConfigurations should be nil, got %v", p.SubnetConfigurations)
		}
	})

	t.Run("NilSafety", func(t *testing.T) {
		nilDNS := vpc.Endpoint{ServiceName: "svc", Type: types.VpcEndpointTypeGateway, RouteTableIDs: []string{"rtb-1"}}
		got := nilDNS.Params()
		checkDnsOptions(t, got.DnsOptions, nilDNS.DNSOptions)
		checkTagSpecs(t, got.TagSpecifications, nilDNS.Tags)
	})
}

func TestEndpointYAMLRoundtrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	ep := vpc.Endpoint{
		ID:                       "vpce-abc123",
		VPCID:                    "vpc-xyz789",
		OwnerID:                  "123456789012",
		ServiceName:              "com.amazonaws.us-east-1.s3",
		Type:                     types.VpcEndpointTypeInterface,
		State:                    types.StateAvailable,
		SubnetIDs:                []string{"subnet-1", "subnet-2"},
		SecurityGroupIDs:         []string{"sg-1"},
		RouteTableIDs:            []string{"rtb-1"},
		NetworkInterfaceIDs:      []string{"eni-1"},
		IPAddressType:            types.IpAddressTypeIpv4,
		IPv4Prefixes:             []vpc.SubnetIPPrefixes{{SubnetID: "subnet-1", IPPrefixes: []string{"10.0.0.0/24"}}},
		IPv6Prefixes:             []vpc.SubnetIPPrefixes{},
		DNSEntries:               []vpc.DNSEntry{{DNSName: "vpce.example.com", HostedZoneID: "Z123"}},
		DNSOptions:               &vpc.DNSOptions{DNSRecordIPType: types.DnsRecordIpTypeIpv4, PrivateDNSOnlyForInboundResolverEndpoint: true, PrivateDNSPreference: "ALL_DOMAINS", PrivateDNSSpecifiedDomains: []string{}},
		PrivateDNSEnabled:        true,
		PolicyDocument:           `{"Version":"2012-10-17"}`,
		ServiceNetworkARN:        "arn:aws:vpc-lattice:us-east-1:123:servicenetwork/sn-x",
		ServiceRegion:            "us-west-2",
		ResourceConfigurationARN: "arn:aws:vpc-lattice:us-east-1:123:resourceconfiguration/rcfg-x",
		CreatedAt:                &createdAt,
		RequesterManaged:         true,
		FailureReason:            "none",
		LastError:                &vpc.EndpointError{Code: "err-code", Message: "err-msg"},
		Tags:                     []vpc.Tag{{Name: "Env", Value: "test"}, {Name: "Owner", Value: "team"}},
	}

	data, err := yaml.Marshal(ep)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	var got vpc.Endpoint
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if !reflect.DeepEqual(ep, got) {
		// Re-marshal the roundtripped value to make diffs readable.
		gotData, _ := yaml.Marshal(got)
		t.Errorf("roundtrip mismatch\noriginal YAML:\n%s\nroundtripped YAML:\n%s", data, gotData)
	}
}

func TestDescribe(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := context.Background()
	client := awsService.EC2(cfg)

	vpcID, subnetID, sgID, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v, err := vpc.NewVPC(ctx, vpcID, vpc.WithClient(client), vpc.WithConfig(cfg))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}
	if err := v.Describe(ctx); err != nil {
		t.Fatalf("Describe: %v", err)
	}

	cfg2 := v.Config
	if cfg2 == nil {
		t.Fatal("Config is nil after Describe")
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
	ctx := awsconfig.ContextWith(context.Background(), cfg)
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v, err := vpc.NewVPC(ctx, vpcID, vpc.WithConfig(cfg), vpc.WithClient(client))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}

	createParams := vpc.Endpoint{
		ServiceName:   "com.amazonaws.us-east-1.s3",
		Type:          types.VpcEndpointTypeGateway,
		RouteTableIDs: []string{rtID},
	}
	endpointID, err := v.CreateEndpoint(ctx, createParams)
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}
	if endpointID == "" {
		t.Fatal("CreateEndpoint returned empty ID")
	}
	defer v.DeleteEndpoint(ctx, endpointID) //nolint:errcheck

	// Verify the endpoint appears in Describe.
	if err := v.Describe(ctx); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	endpointIDs := make([]string, len(v.Config.Endpoints))
	for i, ep := range v.Config.Endpoints {
		endpointIDs[i] = ep.ID
	}
	if !slices.Contains(endpointIDs, endpointID) {
		t.Errorf("endpoints %v does not contain %q", endpointIDs, endpointID)
	}

	// Verify Params() roundtrip: describe the endpoint and convert back to params.
	eps, err := v.DescribeEndpoints(ctx, []string{endpointID})
	if err != nil {
		t.Fatalf("DescribeEndpoints: %v", err)
	}
	if len(eps) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(eps))
	}
	ep := eps[0]
	if ep.ID != endpointID {
		t.Errorf("Endpoint.ID: got %q, want %q", ep.ID, endpointID)
	}
	if ep.VPCID != vpcID {
		t.Errorf("Endpoint.VPCID: got %q, want %q", ep.VPCID, vpcID)
	}

	p := ep.Params()
	if aws.ToString(p.ServiceName) != createParams.ServiceName {
		t.Errorf("Params().ServiceName: got %q, want %q", aws.ToString(p.ServiceName), createParams.ServiceName)
	}
	if p.VpcEndpointType != createParams.Type {
		t.Errorf("Params().VpcEndpointType: got %v, want %v", p.VpcEndpointType, createParams.Type)
	}
	if !slices.Contains(p.RouteTableIds, rtID) {
		t.Errorf("Params().RouteTableIds %v does not contain %q", p.RouteTableIds, rtID)
	}
}

func TestDescribeEndpoints(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	cfg := awstestutil.DefaultAWSConfig()
	ctx := awsconfig.ContextWith(context.Background(), cfg)
	client := awsService.EC2(cfg)

	vpcID, _, _, rtID, cleanup := setupVPC(t, client)
	defer cleanup()

	v, err := vpc.NewVPC(ctx, vpcID, vpc.WithConfig(cfg), vpc.WithClient(client))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}

	// No endpoints yet — list should be empty.
	eps, err := v.DescribeEndpoints(ctx, nil)
	if err != nil {
		t.Fatalf("DescribeEndpoints (empty): %v", err)
	}
	if len(eps) != 0 {
		t.Errorf("expected 0 endpoints before creation, got %d", len(eps))
	}

	// Create an endpoint.
	endpointID, err := v.CreateEndpoint(ctx, vpc.Endpoint{
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
	v, err := vpc.NewVPC(ctx, vpcID, vpc.WithConfig(cfg), vpc.WithClient(client))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}
	endpointID, err := v.CreateEndpoint(ctx, vpc.Endpoint{
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

	v, err := vpc.NewVPC(ctx, vpcID, vpc.WithConfig(cfg), vpc.WithClient(client))
	if err != nil {
		t.Fatalf("NewVPC: %v", err)
	}
	if err := v.DeleteEndpoint(ctx, "vpce-000000000000abcd"); err != nil {
		t.Errorf("unexpected error deleting nonexistent endpoint: %v", err)
	}
}
