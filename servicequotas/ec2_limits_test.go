package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestRulesPerSecurityGroupUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                            errors.New("some err"),
		DescribeSecurityGroupsResponse: nil,
	}

	check := RulesPerSecurityGroupUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestRulesPerSecurityGroupUsage(t *testing.T) {
	testCases := []struct {
		name           string
		securityGroups []types.SecurityGroup
		expectedUsage  []QuotaUsage
	}{
		{
			name:           "WithNoSecurityGroups",
			securityGroups: []types.SecurityGroup{},
			expectedUsage:  []QuotaUsage{},
		},
		{
			name: "WithSecurityGroups",
			securityGroups: []types.SecurityGroup{
				{
					GroupId:             aws.String("somegroupid"),
					IpPermissions:       []types.IpPermission{},
					IpPermissionsEgress: []types.IpPermission{},
				},
				{
					GroupId: aws.String("groupwithrules"),
					IpPermissions: []types.IpPermission{
						{
							FromPort: aws.Int32(0),
							ToPort:   aws.Int32(0),
							UserIdGroupPairs: []types.UserIdGroupPair{
								{
									Description: aws.String("Allow workers to communicate with the control plane."),
									GroupId:     aws.String("sg-0afb91d177e53ae1d"),
									UserId:      aws.String("740679791268"),
								},
							},
							IpRanges: []types.IpRange{
								{
									CidrIp:      aws.String("10.0.0.10/32"),
									Description: aws.String("Rule A"),
								},
								{
									CidrIp:      aws.String("10.0.0.5/32"),
									Description: aws.String("Rule B"),
								},
							},
						},
					},
					IpPermissionsEgress: []types.IpPermission{
						{
							FromPort: aws.Int32(0),
							ToPort:   aws.Int32(0),
							IpRanges: []types.IpRange{
								{
									CidrIp:      aws.String("0.0.0.0/0"),
									Description: aws.String("Rule A"),
								},
							},
						},
					},
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         inboundRulesPerSecGrpName,
					ResourceName: aws.String("somegroupid"),
					Description:  inboundRulesPerSecGrpDesc,
					Usage:        0,
				},
				{
					Name:         outboundRulesPerSecGrpName,
					ResourceName: aws.String("somegroupid"),
					Description:  outboundRulesPerSecGrpDesc,
					Usage:        0,
				},
				{
					Name:         inboundRulesPerSecGrpName,
					ResourceName: aws.String("groupwithrules"),
					Description:  inboundRulesPerSecGrpDesc,
					Usage:        3,
				},
				{
					Name:         outboundRulesPerSecGrpName,
					ResourceName: aws.String("groupwithrules"),
					Description:  outboundRulesPerSecGrpDesc,
					Usage:        1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeSecurityGroupsResponse: &ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: tc.securityGroups,
				},
			}

			check := RulesPerSecurityGroupUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

func TestSecurityGroupsPerENIUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                               errors.New("some err"),
		DescribeNetworkInterfacesResponse: nil,
	}

	check := SecurityGroupsPerENIUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestSecurityGroupsPerENIUsage(t *testing.T) {
	testCases := []struct {
		name              string
		networkInterfaces []types.NetworkInterface
		expectedUsage     []QuotaUsage
	}{
		{
			name:              "WithNoNetworkInterfaces",
			networkInterfaces: []types.NetworkInterface{},
			expectedUsage:     []QuotaUsage{},
		},
		{
			name: "WithNetworkInterfaces",
			networkInterfaces: []types.NetworkInterface{
				{
					NetworkInterfaceId: aws.String("someeni"),
					Groups: []types.GroupIdentifier{
						{
							GroupId:   aws.String("someid"),
							GroupName: aws.String("somename"),
						},
						{
							GroupId:   aws.String("someotherid"),
							GroupName: aws.String("someothername"),
						},
					},
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         secGroupsPerENIName,
					ResourceName: aws.String("someeni"),
					Description:  secGroupsPerENIDesc,
					Usage:        2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeNetworkInterfacesResponse: &ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: tc.networkInterfaces,
				},
			}

			check := SecurityGroupsPerENIUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

func TestSecurityGroupsPerRegionUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                            errors.New("some err"),
		DescribeSecurityGroupsResponse: nil,
	}

	check := SecurityGroupsPerRegionUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestSecurityGroupsPerRegionUsage(t *testing.T) {
	testCases := []struct {
		name           string
		securityGroups []types.SecurityGroup
		expectedUsage  []QuotaUsage
	}{
		{
			name:           "WithNoSecurityGroups",
			securityGroups: []types.SecurityGroup{},
			expectedUsage: []QuotaUsage{
				{
					Name:        securityGroupsPerRegionName,
					Description: securityGroupsPerRegionDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithSecurityGroups",
			securityGroups: []types.SecurityGroup{
				{
					GroupId: aws.String("somegroupid"),
				},
				{
					GroupId: aws.String("anothergroupid"),
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        securityGroupsPerRegionName,
					Description: securityGroupsPerRegionDesc,
					Usage:       2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeSecurityGroupsResponse: &ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: tc.securityGroups,
				},
			}

			check := SecurityGroupsPerRegionUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

func TestStandardInstancesCPUsWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                       errors.New("some err"),
		DescribeInstancesResponse: nil,
	}

	cpus, err := standardInstancesCPUs(mockClient, true)

	assert.Error(t, err)
	assert.Equal(t, int64(0), cpus)
}

func TestStandardInstancesCPUsFilters(t *testing.T) {
	instanceTypeFilter := standardInstanceTypeFilter()
	instanceStateFilter := activeInstanceFilter()

	testCases := []struct {
		name            string
		spotInstances   bool
		expectedFilters []types.Filter
	}{
		{
			name:          "ForSpotInstances",
			spotInstances: true,
			expectedFilters: []types.Filter{
				instanceTypeFilter,
				instanceStateFilter,
				{
					Name:   aws.String("instance-lifecycle"),
					Values: []string{"spot"},
				},
			},
		},
		{
			name:            "ForOnDemandInstances",
			spotInstances:   false,
			expectedFilters: []types.Filter{instanceTypeFilter, instanceStateFilter},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{err: nil, DescribeInstancesResponse: nil}

			cpus, err := standardInstancesCPUs(mockClient, tc.spotInstances)

			assert.NoError(t, err)
			assert.Equal(t, int64(0), cpus)
			assert.Equal(t, mockClient.InstancesFilters, tc.expectedFilters)
		})
	}
}

func TestStandardInstancesCPUs(t *testing.T) {
	mockClient := &mockEC2Client{
		err: nil,
		DescribeInstancesResponse: &ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						{
							InstanceLifecycle: types.InstanceLifecycleTypeSpot,
							CpuOptions: &types.CpuOptions{
								CoreCount:      aws.Int32(4),
								ThreadsPerCore: aws.Int32(2),
							},
						},
					},
				},
				{
					Instances: []types.Instance{
						{
							CpuOptions: &types.CpuOptions{
								CoreCount:      aws.Int32(2),
								ThreadsPerCore: aws.Int32(2),
							},
						},
						{
							CpuOptions: &types.CpuOptions{
								CoreCount:      aws.Int32(4),
								ThreadsPerCore: aws.Int32(2),
							},
						},
					},
				},
			},
		},
	}

	cpus, err := standardInstancesCPUs(mockClient, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), cpus)
}

func TestAvailableIpsPerSubnetUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                     errors.New("some err"),
		DescribeSubnetsResponse: nil,
	}

	check := AvailableIpsPerSubnetUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestAvailableIpsPerSubnetUsageWithInvalidCidrConversion(t *testing.T) {
	mockClient := &mockEC2Client{
		DescribeSubnetsResponse: &ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(4096),
					CidrBlock:               aws.String("invalid-cidr"),
					SubnetId:                aws.String("subnet-id"),
				},
			},
		},
	}
	check := AvailableIpsPerSubnetUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToConvertCidr))
	assert.Nil(t, usage)
}

func TestAvailableIpsPerSubnetUsage(t *testing.T) {
	testCases := []struct {
		name          string
		subnets       []types.Subnet
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoSubnets",
			subnets:       []types.Subnet{},
			expectedUsage: []QuotaUsage{},
		},
		{
			name: "WithSingleSubnet",
			subnets: []types.Subnet{
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(4096),
					CidrBlock:               aws.String("100.10.10.0/20"),
					SubnetId:                aws.String("subnet-id"),
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         availableIPsPerSubnetName,
					ResourceName: aws.String("subnet-id"),
					Description:  availableIPsPerSubnetDesc,
					Usage:        float64(0),
					Quota:        float64(4096),
				},
			},
		},
		{
			name: "WithMultipleSubnets",
			subnets: []types.Subnet{
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(4096),
					CidrBlock:               aws.String("100.10.10.0/20"),
					SubnetId:                aws.String("subnet-id-1"),
				},
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(0),
					CidrBlock:               aws.String("100.10.10.0/21"),
					SubnetId:                aws.String("subnet-id-2"),
				},
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(100),
					CidrBlock:               aws.String("100.10.10.0/21"),
					SubnetId:                aws.String("subnet-id-2"),
				},
				{
					AvailabilityZone:        aws.String("eu-west-1"),
					AvailableIpAddressCount: aws.Int32(1024),
					CidrBlock:               aws.String("100.10.10.0/22"),
					SubnetId:                aws.String("subnet-id-3"),
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         availableIPsPerSubnetName,
					ResourceName: aws.String("subnet-id-1"),
					Description:  availableIPsPerSubnetDesc,
					Usage:        float64(0),
					Quota:        float64(4096),
				},
				{
					Name:         availableIPsPerSubnetName,
					ResourceName: aws.String("subnet-id-2"),
					Description:  availableIPsPerSubnetDesc,
					Usage:        float64(2048),
					Quota:        float64(2048),
				},
				{
					Name:         availableIPsPerSubnetName,
					ResourceName: aws.String("subnet-id-2"),
					Description:  availableIPsPerSubnetDesc,
					Usage:        float64(1948),
					Quota:        float64(2048),
				},
				{
					Name:         availableIPsPerSubnetName,
					ResourceName: aws.String("subnet-id-3"),
					Description:  availableIPsPerSubnetDesc,
					Usage:        float64(0),
					Quota:        float64(1024),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeSubnetsResponse: &ec2.DescribeSubnetsOutput{
					Subnets: tc.subnets,
				},
			}

			check := AvailableIpsPerSubnetUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}
