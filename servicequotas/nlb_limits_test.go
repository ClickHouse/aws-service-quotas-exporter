package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/stretchr/testify/assert"
)

func (m *mockELBV2Client) DescribeLoadBalancersPages(input *elbv2.DescribeLoadBalancersInput, fn func(*elbv2.DescribeLoadBalancersOutput, bool) bool) error {
	fn(m.DescribeLoadBalancersResponse, true)
	return m.err
}

func TestNLBsPerRegionUsageWithError(t *testing.T) {
	mockClient := &mockELBV2Client{
		err:                           errors.New("some err"),
		DescribeLoadBalancersResponse: nil,
	}

	check := NLBsPerRegionUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestNLBsPerRegionUsage(t *testing.T) {
	testCases := []struct {
		name          string
		loadBalancers []*elbv2.LoadBalancer
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoLoadBalancers",
			loadBalancers: []*elbv2.LoadBalancer{},
			expectedUsage: []QuotaUsage{
				{
					Name:        nlbsPerRegionName,
					Description: nlbsPerRegionDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMixedTypes",
			loadBalancers: []*elbv2.LoadBalancer{
				{Type: aws.String(elbv2.LoadBalancerTypeEnumNetwork)},
				{Type: aws.String(elbv2.LoadBalancerTypeEnumNetwork)},
				{Type: aws.String(elbv2.LoadBalancerTypeEnumApplication)},
				{Type: aws.String("gateway")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        nlbsPerRegionName,
					Description: nlbsPerRegionDesc,
					Usage:       2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockELBV2Client{
				err: nil,
				DescribeLoadBalancersResponse: &elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: tc.loadBalancers,
				},
			}

			check := NLBsPerRegionUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}
