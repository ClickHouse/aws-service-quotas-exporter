package servicequotas

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ec2API is the subset of the EC2 client used across the EC2-backed usage
// checks (ec2, vpc and vpc endpoint limits). It is satisfied by *ec2.Client
// and by the EC2 paginators' *APIClient interfaces.
type ec2API interface {
	DescribeSecurityGroups(context.Context, *ec2.DescribeSecurityGroupsInput, ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeSubnets(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeVpcs(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeVpcEndpoints(context.Context, *ec2.DescribeVpcEndpointsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DescribeAddresses(context.Context, *ec2.DescribeAddressesInput, ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
}

// Not all quota limits here are reported under "ec2", but all of the
// usage checks are using the ec2 service
const (
	inboundRulesPerSecGrpName = "inbound_rules_per_security_group"
	inboundRulesPerSecGrpDesc = "inbound rules per security group"

	outboundRulesPerSecGrpName = "outbound_rules_per_security_group"
	outboundRulesPerSecGrpDesc = "outbound rules per security group"

	secGroupsPerENIName = "security_groups_per_network_interface"
	secGroupsPerENIDesc = "security groups per network interface"

	securityGroupsPerRegionName = "security_groups_per_region"
	securityGroupsPerRegionDesc = "security groups per region"

	spotInstanceRequestsName = "spot_instance_requests"
	spotInstanceRequestsDesc = "spot instance requests"

	onDemandInstanceRequestsName = "ondemand_instance_requests"
	onDemandInstanceRequestsDesc = "ondemand instance requests"

	availableIPsPerSubnetName = "available_ips_per_subnet"
	availableIPsPerSubnetDesc = "available IPs per subnet"
)

// RulesPerSecurityGroupUsageCheck implements the UsageCheck interface
// for rules per security group
type RulesPerSecurityGroupUsageCheck struct {
	client ec2API
}

// Usage returns the usage for each security group ID with the usage
// value being the sum of their inbound and outbound rules or an error
func (c *RulesPerSecurityGroupUsageCheck) Usage() ([]QuotaUsage, error) {
	quotaUsages := []QuotaUsage{}

	params := &ec2.DescribeSecurityGroupsInput{}
	paginator := ec2.NewDescribeSecurityGroupsPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, group := range page.SecurityGroups {
			var inboundRules int
			var outboundRules int

			tags := ec2TagsToQuotaUsageTags(group.Tags)

			for _, rule := range group.IpPermissions {
				inboundRules += len(rule.IpRanges)
				inboundRules += len(rule.UserIdGroupPairs)
			}

			inboundUsage := QuotaUsage{
				Name:         inboundRulesPerSecGrpName,
				ResourceName: group.GroupId,
				Description:  inboundRulesPerSecGrpDesc,
				Usage:        float64(inboundRules),
				Tags:         tags,
			}

			for _, rule := range group.IpPermissionsEgress {
				outboundRules += len(rule.IpRanges)
				outboundRules += len(rule.UserIdGroupPairs)
			}

			outboundUsage := QuotaUsage{
				Name:         outboundRulesPerSecGrpName,
				ResourceName: group.GroupId,
				Description:  outboundRulesPerSecGrpDesc,
				Usage:        float64(outboundRules),
				Tags:         tags,
			}

			quotaUsages = append(quotaUsages, []QuotaUsage{inboundUsage, outboundUsage}...)
		}
	}

	return quotaUsages, nil
}

// SecurityGroupsPerENIUsageCheck implements the UsageCheck interface
// for security groups per ENI
type SecurityGroupsPerENIUsageCheck struct {
	client ec2API
}

// Usage returns usage for each Elastic Network Interface ID with the
// usage value being the number of security groups for each ENI or an
// error
func (c *SecurityGroupsPerENIUsageCheck) Usage() ([]QuotaUsage, error) {
	quotaUsages := []QuotaUsage{}

	params := &ec2.DescribeNetworkInterfacesInput{}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, eni := range page.NetworkInterfaces {
			usage := QuotaUsage{
				Name:         secGroupsPerENIName,
				ResourceName: eni.NetworkInterfaceId,
				Description:  secGroupsPerENIDesc,
				Usage:        float64(len(eni.Groups)),
				Tags:         ec2TagsToQuotaUsageTags(eni.TagSet),
			}
			quotaUsages = append(quotaUsages, usage)
		}
	}

	return quotaUsages, nil
}

// SecurityGroupsPerRegionUsageCheck implements the UsageCheck interface
// for security groups per region
type SecurityGroupsPerRegionUsageCheck struct {
	client ec2API
}

// Usage returns usage for security groups per region as the number of
// all security groups for the region specified with `cfgs` or an error
func (c *SecurityGroupsPerRegionUsageCheck) Usage() ([]QuotaUsage, error) {
	numGroups := 0

	params := &ec2.DescribeSecurityGroupsInput{}
	paginator := ec2.NewDescribeSecurityGroupsPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}
		numGroups += len(page.SecurityGroups)
	}

	usage := []QuotaUsage{
		{
			Name:        securityGroupsPerRegionName,
			Description: securityGroupsPerRegionDesc,
			Usage:       float64(numGroups),
		},
	}
	return usage, nil
}

func standardInstanceTypeFilter() types.Filter {
	return types.Filter{
		Name:   aws.String("instance-type"),
		Values: []string{"a*", "c*", "d*", "h*", "i*", "m*", "r*", "t*", "z*"},
	}
}

func activeInstanceFilter() types.Filter {
	return types.Filter{
		Name:   aws.String("instance-state-name"),
		Values: []string{"pending", "running"},
	}
}

// standardInstancesCPUs returns the number of vCPUs for all standard
// (A, C, D, H, I, M, R, T, Z) EC2 instances
// Note that we are working out the number of vCPUs for each instance
// here because instances can have custom CPU options specified during
// launch. More information can be found at
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-optimize-cpu.html
func standardInstancesCPUs(ec2Service ec2API, spotInstances bool) (int64, error) {
	var totalvCPUs int64
	instanceTypeFilter := standardInstanceTypeFilter()
	instanceStateFilter := activeInstanceFilter()
	filters := []types.Filter{instanceTypeFilter, instanceStateFilter}

	// According to the AWS docs we should be able to filter
	// "scheduled" instances as well, but that does not work so we
	// are using filters only for the spot instances
	if spotInstances {
		spotFilter := types.Filter{
			Name:   aws.String("instance-lifecycle"),
			Values: []string{"spot"},
		}
		filters = append(filters, spotFilter)
	}

	params := &ec2.DescribeInstancesInput{Filters: filters}
	paginator := ec2.NewDescribeInstancesPaginator(ec2Service, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return 0, err
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				// InstanceLifecycle is empty for On-Demand instances
				if !spotInstances && instance.InstanceLifecycle != "" {
					continue
				}

				cpuOptions := instance.CpuOptions
				if cpuOptions != nil && cpuOptions.CoreCount != nil && cpuOptions.ThreadsPerCore != nil {
					numvCPUs := int64(*cpuOptions.CoreCount) * int64(*cpuOptions.ThreadsPerCore)
					totalvCPUs += numvCPUs
				}
			}
		}
	}

	return totalvCPUs, nil
}

// StandardSpotInstanceRequestsUsageCheck implements the UsageCheck interface
// for standard spot instance requests
type StandardSpotInstanceRequestsUsageCheck struct {
	client ec2API
}

// Usage returns vCPU usage for all standard (A, C, D, H, I, M, R, T,
// Z) spot instance requests and usage or an error
// vCPUs are returned instead of the number of images due to the
// service quota reporting the number of vCPUs
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-spot-limits.html
func (c *StandardSpotInstanceRequestsUsageCheck) Usage() ([]QuotaUsage, error) {
	cpus, err := standardInstancesCPUs(c.client, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	usage := []QuotaUsage{
		{
			Name:        spotInstanceRequestsName,
			Description: spotInstanceRequestsDesc,
			Usage:       float64(cpus),
		},
	}
	return usage, nil
}

// RunningOnDemandStandardInstancesUsageCheck implements the UsageCheck interface
// for standard on-demand instances
type RunningOnDemandStandardInstancesUsageCheck struct {
	client ec2API
}

// Usage returns vCPU usage for all running on-demand standard (A, C,
// D, H, I, M, R, T, Z) instances or an error vCPUs are returned instead
// of the number of images due to the service quota reporting the number
// of vCPUs
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-spot-limits.html
func (c *RunningOnDemandStandardInstancesUsageCheck) Usage() ([]QuotaUsage, error) {
	cpus, err := standardInstancesCPUs(c.client, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	usage := []QuotaUsage{
		{
			Name:        onDemandInstanceRequestsName,
			Description: onDemandInstanceRequestsDesc,
			Usage:       float64(cpus),
		},
	}
	return usage, nil
}

// AvailableIpsPerSubnetUsageCheck implements the UsageCheckInterface
// for available IPs per subnet
type AvailableIpsPerSubnetUsageCheck struct {
	client ec2API
}

// Usage returns the usage for each subnet ID with the usage value
// being the number of available IPv4 addresses in that subnet or
// an error
// Note that the Description of the resource here is constructed
// using `availableIPsPerSubnetDesc` defined previously as well as
// the subnet's CIDR block
func (c *AvailableIpsPerSubnetUsageCheck) Usage() ([]QuotaUsage, error) {
	availabilityInfos := []QuotaUsage{}

	params := &ec2.DescribeSubnetsInput{}
	paginator := ec2.NewDescribeSubnetsPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}

		for _, subnet := range page.Subnets {
			cidrBlock := *subnet.CidrBlock
			blockedBits, err := strconv.Atoi(cidrBlock[len(cidrBlock)-2:])
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrFailedToConvertCidr, err)
			}
			maxNumOfIPs := math.Pow(2, 32-float64(blockedBits))
			usage := float64(maxNumOfIPs - float64(*subnet.AvailableIpAddressCount))
			availabilityInfo := QuotaUsage{
				Name:         availableIPsPerSubnetName,
				ResourceName: subnet.SubnetId,
				Description:  availableIPsPerSubnetDesc,
				Usage:        usage,
				Quota:        float64(maxNumOfIPs),
				Tags:         ec2TagsToQuotaUsageTags(subnet.Tags),
			}
			availabilityInfos = append(availabilityInfos, availabilityInfo)
		}
	}

	return availabilityInfos, nil
}

func ec2TagsToQuotaUsageTags(tags []types.Tag) map[string]string {
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
