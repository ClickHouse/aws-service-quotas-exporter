package servicequotas

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
)

type mockAutoScalingClient struct {
	err                               error
	DescribeAutoScalingGroupsResponse *autoscaling.DescribeAutoScalingGroupsOutput
}

func (m *mockAutoScalingClient) DescribeAutoScalingGroups(_ context.Context, _ *autoscaling.DescribeAutoScalingGroupsInput, _ ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.DescribeAutoScalingGroupsResponse == nil {
		return &autoscaling.DescribeAutoScalingGroupsOutput{}, nil
	}
	return m.DescribeAutoScalingGroupsResponse, nil
}
