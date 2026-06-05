package servicequotas

import (
	"context"
	"fmt"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

const (
	nlbsPerRegionName = "nlbs_per_region"
	nlbsPerRegionDesc = "Network Load Balancers per region"
)

// elbv2API is the subset of the ELBv2 client used by the NLB usage check.
type elbv2API interface {
	DescribeLoadBalancers(context.Context, *elbv2.DescribeLoadBalancersInput, ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
}

// NLBsPerRegionUsageCheck implements the UsageCheck interface for
// Network Load Balancers per region (quota L-69A177A2).
type NLBsPerRegionUsageCheck struct {
	client elbv2API
}

// Usage returns the number of Network Load Balancers in the region.
func (c *NLBsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	numNLBs := 0

	params := &elbv2.DescribeLoadBalancersInput{}
	paginator := elbv2.NewDescribeLoadBalancersPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, lb := range page.LoadBalancers {
			if lb.Type == types.LoadBalancerTypeEnumNetwork {
				numNLBs++
			}
		}
	}

	return []QuotaUsage{
		{
			Name:        nlbsPerRegionName,
			Description: nlbsPerRegionDesc,
			Usage:       float64(numNLBs),
		},
	}, nil
}
