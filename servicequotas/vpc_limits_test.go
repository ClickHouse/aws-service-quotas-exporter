package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestVpcsPerRegionUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                  errors.New("some err"),
		DescribeVpcsResponse: nil,
	}

	check := VpcsPerRegionUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestVpcsPerRegionUsage(t *testing.T) {
	testCases := []struct {
		name          string
		vpcs          []types.Vpc
		expectedUsage []QuotaUsage
	}{
		{
			name: "WithNoVpcs",
			vpcs: []types.Vpc{},
			expectedUsage: []QuotaUsage{
				{
					Name:        vpcsPerRegionName,
					Description: vpcsPerRegionDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMultipleVpcs",
			vpcs: []types.Vpc{
				{VpcId: aws.String("vpc-aaa")},
				{VpcId: aws.String("vpc-bbb")},
				{VpcId: aws.String("vpc-ccc")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        vpcsPerRegionName,
					Description: vpcsPerRegionDesc,
					Usage:       3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeVpcsResponse: &ec2.DescribeVpcsOutput{
					Vpcs: tc.vpcs,
				},
			}

			check := VpcsPerRegionUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

func TestEIPsPerRegionUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                       errors.New("some err"),
		DescribeAddressesResponse: nil,
	}

	check := EIPsPerRegionUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestEIPsPerRegionUsage(t *testing.T) {
	testCases := []struct {
		name          string
		addresses     []types.Address
		expectedUsage []QuotaUsage
	}{
		{
			name:      "WithNoEIPs",
			addresses: []types.Address{},
			expectedUsage: []QuotaUsage{
				{
					Name:        eipsPerRegionName,
					Description: eipsPerRegionDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMultipleEIPs",
			addresses: []types.Address{
				{AllocationId: aws.String("eipalloc-1")},
				{AllocationId: aws.String("eipalloc-2")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        eipsPerRegionName,
					Description: eipsPerRegionDesc,
					Usage:       2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeAddressesResponse: &ec2.DescribeAddressesOutput{
					Addresses: tc.addresses,
				},
			}

			check := EIPsPerRegionUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}
