package servicequotas

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

const (
	vpcsPerRegionName = "vpcs_per_region"
	vpcsPerRegionDesc = "VPCs per region"

	eipsPerRegionName = "eips_per_region"
	eipsPerRegionDesc = "EC2-VPC Elastic IP addresses per region"
)

// VpcsPerRegionUsageCheck implements the UsageCheck interface for VPCs per region.
type VpcsPerRegionUsageCheck struct {
	client ec2iface.EC2API
}

// Usage returns the number of VPCs in the region.
func (c *VpcsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	numVpcs := 0

	params := &ec2.DescribeVpcsInput{}
	err := c.client.DescribeVpcsPages(params,
		func(page *ec2.DescribeVpcsOutput, lastPage bool) bool {
			if page != nil {
				numVpcs += len(page.Vpcs)
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
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
	client ec2iface.EC2API
}

// Usage returns the number of EC2-VPC Elastic IPs allocated in the region.
func (c *EIPsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	output, err := c.client.DescribeAddresses(&ec2.DescribeAddressesInput{})
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
