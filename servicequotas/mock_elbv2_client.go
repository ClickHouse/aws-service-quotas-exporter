package servicequotas

import (
	"context"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type mockELBV2Client struct {
	err                           error
	DescribeLoadBalancersResponse *elbv2.DescribeLoadBalancersOutput
}

func (m *mockELBV2Client) DescribeLoadBalancers(_ context.Context, _ *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeLoadBalancersResponse == nil {
		return &elbv2.DescribeLoadBalancersOutput{}, nil
	}
	return m.DescribeLoadBalancersResponse, nil
}
