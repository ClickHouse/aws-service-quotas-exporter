package servicequotas

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

const (
	nlbsPerRegionName = "nlbs_per_region"
	nlbsPerRegionDesc = "Network Load Balancers per region"
)

// NLBsPerRegionUsageCheck implements the UsageCheck interface for
// Network Load Balancers per region (quota L-69A177A2).
type NLBsPerRegionUsageCheck struct {
	client elbv2iface.ELBV2API
}

// Usage returns the number of Network Load Balancers in the region.
func (c *NLBsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	numNLBs := 0

	params := &elbv2.DescribeLoadBalancersInput{}
	err := c.client.DescribeLoadBalancersPages(params,
		func(page *elbv2.DescribeLoadBalancersOutput, lastPage bool) bool {
			if page != nil {
				for _, lb := range page.LoadBalancers {
					if lb.Type != nil && *lb.Type == elbv2.LoadBalancerTypeEnumNetwork {
						numNLBs++
					}
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	return []QuotaUsage{
		{
			Name:        nlbsPerRegionName,
			Description: nlbsPerRegionDesc,
			Usage:       float64(numNLBs),
		},
	}, nil
}
