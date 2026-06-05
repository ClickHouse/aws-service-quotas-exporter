package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/stretchr/testify/assert"
)

func TestASGUsageCheckWithError(t *testing.T) {
	mockClient := &mockAutoScalingClient{
		err:                               errors.New("some err"),
		DescribeAutoScalingGroupsResponse: nil,
	}

	check := ASGUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestASGUsageCheck(t *testing.T) {
	mockClient := &mockAutoScalingClient{
		err: nil,
		DescribeAutoScalingGroupsResponse: &autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []types.AutoScalingGroup{
				{
					AutoScalingGroupName: aws.String("asg1"),
					Instances: []types.Instance{
						{LifecycleState: types.LifecycleState("Terminating")},
						{LifecycleState: types.LifecycleState("Terminating:Wait")},
						{LifecycleState: types.LifecycleState("Terminating:Proceed")},
						{LifecycleState: types.LifecycleState("Terminated")},
						{LifecycleState: types.LifecycleState("Detaching")},
						{LifecycleState: types.LifecycleState("Detached")},
						{LifecycleState: types.LifecycleState("InService")},
						{LifecycleState: types.LifecycleState("Pending")},
					},
					MaxSize: aws.Int32(7),
				},
				{
					AutoScalingGroupName: aws.String("asg2"),
					Instances:            []types.Instance{},
					MaxSize:              aws.Int32(3),
				},
				{
					AutoScalingGroupName: aws.String("asg3"),
					Instances: []types.Instance{
						{LifecycleState: types.LifecycleState("InService")},
						{LifecycleState: types.LifecycleState("InService")},
						{LifecycleState: types.LifecycleState("Pending")},
					},
					MaxSize: aws.Int32(10),
				},
			},
		},
	}

	check := ASGUsageCheck{mockClient}
	usage, err := check.Usage()

	expectedUsage := []QuotaUsage{
		{
			Name:         numInstancesPerASGName,
			ResourceName: aws.String("asg1"),
			Description:  numInstancesPerASGDescription,
			Usage:        float64(2),
			Quota:        float64(7),
		},
		{
			Name:         numInstancesPerASGName,
			ResourceName: aws.String("asg2"),
			Description:  numInstancesPerASGDescription,
			Usage:        float64(0),
			Quota:        float64(3),
		},
		{
			Name:         numInstancesPerASGName,
			ResourceName: aws.String("asg3"),
			Description:  numInstancesPerASGDescription,
			Usage:        float64(3),
			Quota:        float64(10),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, expectedUsage, usage)
}
