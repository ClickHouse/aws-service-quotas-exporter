package servicequotas

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

const (
	nlbsPerRegionName = "nlbs_per_region"
	nlbsPerRegionDesc = "Network Load Balancers per region"

	ipsPerNLBAZName = "ips_per_nlb_availability_zone"
	ipsPerNLBAZDesc = "IPv4 addresses per Network Load Balancer Availability Zone"

	// maxIPsPerNLBAZ is the most IPv4 addresses a Network Load Balancer
	// can have in one Availability Zone: its primary address, and up to 7
	// secondary ones from the secondary_ips.auto_assigned.per_subnet
	// load balancer attribute.
	maxIPsPerNLBAZ = 8

	// The description of a load balancer's network interface is "ELB "
	// followed by the load balancer's ID, which for a Network Load
	// Balancer is "net/<name>/<id>".
	elbENIDescriptionPrefix = "ELB "
	nlbIDPrefix             = "net/"

	availabilityZoneLabel = "availability_zone"
	ipLabel               = "ip"
	primaryLabel          = "primary"
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

// IPsPerNLBAvailabilityZoneUsageCheck implements the UsageCheck interface
// for the IPv4 addresses that Network Load Balancers use, per
// Availability Zone. The number of addresses caps the connections a load
// balancer can open to a target when it does not preserve the client IP
// (about 55,000 per address). No CloudWatch metric reports the
// addresses, and DescribeLoadBalancers does not return the secondary
// ones.
type IPsPerNLBAvailabilityZoneUsageCheck struct {
	client ec2API
}

// Usage returns one usage, with the value 1, for each private IPv4
// address on the network interfaces of each Network Load Balancer, or an
// error. The address, whether it is the interface's primary one and the
// Availability Zone are Labels, so the usages of a load balancer in a
// zone add up to its number of addresses there. The ResourceName is the
// load balancer's ID as in its ARN and its CloudWatch LoadBalancer
// dimension ("net/<name>/<id>").
func (c *IPsPerNLBAvailabilityZoneUsageCheck) Usage() ([]QuotaUsage, error) {
	quotaUsages := []QuotaUsage{}

	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("interface-type"),
				Values: []string{string(ec2types.NetworkInterfaceTypeNetworkLoadBalancer)},
			},
		},
	}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, eni := range page.NetworkInterfaces {
			loadBalancer := strings.TrimPrefix(aws.ToString(eni.Description), elbENIDescriptionPrefix)
			zone := aws.ToString(eni.AvailabilityZone)
			if !strings.HasPrefix(loadBalancer, nlbIDPrefix) || zone == "" {
				continue
			}

			for _, address := range eni.PrivateIpAddresses {
				if address.PrivateIpAddress == nil {
					continue
				}

				quotaUsages = append(quotaUsages, QuotaUsage{
					Name:         ipsPerNLBAZName,
					ResourceName: aws.String(loadBalancer),
					Description:  ipsPerNLBAZDesc,
					Usage:        1,
					Quota:        maxIPsPerNLBAZ,
					Labels: map[string]string{
						availabilityZoneLabel: zone,
						ipLabel:               *address.PrivateIpAddress,
						primaryLabel:          strconv.FormatBool(aws.ToBool(address.Primary)),
					},
				})
			}
		}
	}

	return quotaUsages, nil
}
