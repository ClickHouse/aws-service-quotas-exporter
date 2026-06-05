package servicequotas

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	vpcsPerRegionName = "vpcs_per_region"
	vpcsPerRegionDesc = "VPCs per region"

	eipsPerRegionName = "eips_per_region"
	eipsPerRegionDesc = "EC2-VPC Elastic IP addresses per region"
)

// VpcsPerRegionUsageCheck implements the UsageCheck interface for VPCs per region.
type VpcsPerRegionUsageCheck struct {
	client ec2API
}

// Usage returns the number of VPCs in the region.
func (c *VpcsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	numVpcs := 0

	params := &ec2.DescribeVpcsInput{}
	paginator := ec2.NewDescribeVpcsPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}
		numVpcs += len(page.Vpcs)
	}

	return []QuotaUsage{
		{
			Name:        vpcsPerRegionName,
			Description: vpcsPerRegionDesc,
			Usage:       float64(numVpcs),
		},
	}, nil
}

// EIPsPerRegionUsageCheck implements the UsageCheck interface for
// EC2-VPC Elastic IP addresses per region.
type EIPsPerRegionUsageCheck struct {
	client ec2API
}

// Usage returns the number of EC2-VPC Elastic IPs allocated in the region.
func (c *EIPsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	output, err := c.client.DescribeAddresses(context.TODO(), &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	return []QuotaUsage{
		{
			Name:        eipsPerRegionName,
			Description: eipsPerRegionDesc,
			Usage:       float64(len(output.Addresses)),
		},
	}, nil
}
