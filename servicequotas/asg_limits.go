package servicequotas

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
)

const (
	numInstancesPerASGName        = "instances_per_asg"
	numInstancesPerASGDescription = "instances per ASG"
)

// autoscalingAPI is the subset of the Auto Scaling client used by the ASG
// usage check.
type autoscalingAPI interface {
	DescribeAutoScalingGroups(context.Context, *autoscaling.DescribeAutoScalingGroupsInput, ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

// ASGUsageCheck implements the UsageCheckInterface for VMs per
// autoscaling group
type ASGUsageCheck struct {
	client autoscalingAPI
}

// Usage returns usage per auto scaling group - the maximum number of
// instances per ASG and the current number of "running" instances per
// ASG.
func (c *ASGUsageCheck) Usage() ([]QuotaUsage, error) {
	quotaUsages := []QuotaUsage{}

	params := &autoscaling.DescribeAutoScalingGroupsInput{}
	paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, asg := range page.AutoScalingGroups {
			numRunningInstances := 0
			for _, instance := range asg.Instances {
				if isRunning(instance) {
					numRunningInstances++
				}
			}

			quotaUsage := QuotaUsage{
				Name:         numInstancesPerASGName,
				ResourceName: asg.AutoScalingGroupName,
				Description:  numInstancesPerASGDescription,
				Usage:        float64(numRunningInstances),
				Quota:        float64(*asg.MaxSize),
				Tags:         autoscalingTagsToQuotaUsageTags(asg.Tags),
			}
			quotaUsages = append(quotaUsages, quotaUsage)
		}
	}

	return quotaUsages, nil
}

func isRunning(instance types.Instance) bool {
	notRunningStates := map[string]bool{
		"Terminating":         true,
		"Terminating:Wait":    true,
		"Terminating:Proceed": true,
		"Terminated":          true,
		"Detaching":           true,
		"Detached":            true,
	}

	_, isNotRunning := notRunningStates[string(instance.LifecycleState)]
	return !isNotRunning
}

func autoscalingTagsToQuotaUsageTags(tags []types.TagDescription) map[string]string {
	length := len(tags)
	if length == 0 {
		return nil
	}

	out := make(map[string]string, length)
	for _, tag := range tags {
		out[ToPrometheusNamingFormat(*tag.Key)] = *tag.Value
	}

	return out
}
