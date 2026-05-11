package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

func (m *mockEC2Client) DescribeVpcsPages(input *ec2.DescribeVpcsInput, fn func(*ec2.DescribeVpcsOutput, bool) bool) error {
	fn(m.DescribeVpcsResponse, true)
	return m.err
}

func (m *mockEC2Client) DescribeAddresses(input *ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.DescribeAddressesResponse, nil
}

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
		vpcs          []*ec2.Vpc
		expectedUsage []QuotaUsage
	}{
		{
			name: "WithNoVpcs",
			vpcs: []*ec2.Vpc{},
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
			vpcs: []*ec2.Vpc{
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
		addresses     []*ec2.Address
		expectedUsage []QuotaUsage
	}{
		{
			name:      "WithNoEIPs",
			addresses: []*ec2.Address{},
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
			addresses: []*ec2.Address{
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
