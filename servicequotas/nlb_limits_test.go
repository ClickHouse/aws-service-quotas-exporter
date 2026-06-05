package servicequotas

import (
	"errors"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
)

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
		loadBalancers []types.LoadBalancer
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoLoadBalancers",
			loadBalancers: []types.LoadBalancer{},
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
			loadBalancers: []types.LoadBalancer{
				{Type: types.LoadBalancerTypeEnumNetwork},
				{Type: types.LoadBalancerTypeEnumNetwork},
				{Type: types.LoadBalancerTypeEnumApplication},
				{Type: types.LoadBalancerTypeEnum("gateway")},
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
