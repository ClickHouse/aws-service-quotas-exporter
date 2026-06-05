package servicequotas

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2Client struct {
	err                               error
	DescribeSecurityGroupsResponse    *ec2.DescribeSecurityGroupsOutput
	DescribeNetworkInterfacesResponse *ec2.DescribeNetworkInterfacesOutput
	InstancesFilters                  []types.Filter
	DescribeInstancesResponse         *ec2.DescribeInstancesOutput
	DescribeSubnetsResponse           *ec2.DescribeSubnetsOutput
	DescribeVpcEndpointsResponse      *ec2.DescribeVpcEndpointsOutput
	DescribeVpcsResponse              *ec2.DescribeVpcsOutput
	DescribeAddressesResponse         *ec2.DescribeAddressesOutput
}

func (m *mockEC2Client) DescribeSecurityGroups(_ context.Context, _ *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeSecurityGroupsResponse == nil {
		return &ec2.DescribeSecurityGroupsOutput{}, nil
	}
	return m.DescribeSecurityGroupsResponse, nil
}

func (m *mockEC2Client) DescribeNetworkInterfaces(_ context.Context, _ *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeNetworkInterfacesResponse == nil {
		return &ec2.DescribeNetworkInterfacesOutput{}, nil
	}
	return m.DescribeNetworkInterfacesResponse, nil
}

func (m *mockEC2Client) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	m.InstancesFilters = input.Filters
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeInstancesResponse == nil {
		return &ec2.DescribeInstancesOutput{}, nil
	}
	return m.DescribeInstancesResponse, nil
}

func (m *mockEC2Client) DescribeSubnets(_ context.Context, _ *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeSubnetsResponse == nil {
		return &ec2.DescribeSubnetsOutput{}, nil
	}
	return m.DescribeSubnetsResponse, nil
}

func (m *mockEC2Client) DescribeVpcs(_ context.Context, _ *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeVpcsResponse == nil {
		return &ec2.DescribeVpcsOutput{}, nil
	}
	return m.DescribeVpcsResponse, nil
}

func (m *mockEC2Client) DescribeVpcEndpoints(_ context.Context, _ *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeVpcEndpointsResponse == nil {
		return &ec2.DescribeVpcEndpointsOutput{}, nil
	}
	return m.DescribeVpcEndpointsResponse, nil
}

func (m *mockEC2Client) DescribeAddresses(_ context.Context, _ *ec2.DescribeAddressesInput, _ ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeAddressesResponse == nil {
		return &ec2.DescribeAddressesOutput{}, nil
	}
	return m.DescribeAddressesResponse, nil
}
