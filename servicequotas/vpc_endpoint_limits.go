package servicequotas

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

const (
	interfaceVpcEndpointsPerVpcName = "interface_vpc_endpoints_per_vpc"
	interfaceVpcEndpointsPerVpcDesc = "interface VPC endpoints per VPC"

	resourceVpcEndpointsPerVpcName = "resource_vpc_endpoints_per_vpc"
	resourceVpcEndpointsPerVpcDesc = "resource VPC endpoints per VPC"

	serviceNetworkVpcEndpointsPerVpcName = "service_network_vpc_endpoints_per_vpc"
	serviceNetworkVpcEndpointsPerVpcDesc = "service network VPC endpoints per VPC"

	// VPC endpoint types not yet in the SDK as constants
	vpcEndpointTypeResource       = "Resource"
	vpcEndpointTypeServiceNetwork = "ServiceNetwork"
)

// vpcEndpointsByVpcAndType counts VPC endpoints grouped by VPC ID and type.
func vpcEndpointsByVpcAndType(client ec2iface.EC2API) (map[string]map[string]int, error) {
	// map[vpcId]map[endpointType]count
	counts := make(map[string]map[string]int)

	params := &ec2.DescribeVpcEndpointsInput{}
	err := client.DescribeVpcEndpointsPages(params,
		func(page *ec2.DescribeVpcEndpointsOutput, lastPage bool) bool {
			if page != nil {
				for _, endpoint := range page.VpcEndpoints {
					vpcID := *endpoint.VpcId
					endpointType := *endpoint.VpcEndpointType

					if _, ok := counts[vpcID]; !ok {
						counts[vpcID] = make(map[string]int)
					}
					counts[vpcID][endpointType]++
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	return counts, nil
}

// InterfaceVpcEndpointsPerVpcUsageCheck implements the UsageCheck interface
// for Interface and GatewayLoadBalancer VPC endpoints per VPC (quota L-29B6F2EB).
type InterfaceVpcEndpointsPerVpcUsageCheck struct {
	client ec2iface.EC2API
}

// Usage returns the usage of Interface + GatewayLoadBalancer VPC endpoints per VPC.
func (c *InterfaceVpcEndpointsPerVpcUsageCheck) Usage() ([]QuotaUsage, error) {
	counts, err := vpcEndpointsByVpcAndType(c.client)
	if err != nil {
		return nil, err
	}

	var usages []QuotaUsage
	for vpcID, typeCounts := range counts {
		count := typeCounts[ec2.VpcEndpointTypeInterface] + typeCounts[ec2.VpcEndpointTypeGatewayLoadBalancer]
		if count > 0 {
			id := vpcID
			usages = append(usages, QuotaUsage{
				Name:         interfaceVpcEndpointsPerVpcName,
				ResourceName: &id,
				Description:  interfaceVpcEndpointsPerVpcDesc,
				Usage:        float64(count),
			})
		}
	}

	return usages, nil
}

// ResourceVpcEndpointsPerVpcUsageCheck implements the UsageCheck interface
// for Resource VPC endpoints per VPC (quota L-CA6CC422).
type ResourceVpcEndpointsPerVpcUsageCheck struct {
	client ec2iface.EC2API
}

// Usage returns the usage of Resource VPC endpoints per VPC.
func (c *ResourceVpcEndpointsPerVpcUsageCheck) Usage() ([]QuotaUsage, error) {
	counts, err := vpcEndpointsByVpcAndType(c.client)
	if err != nil {
		return nil, err
	}

	var usages []QuotaUsage
	for vpcID, typeCounts := range counts {
		count := typeCounts[vpcEndpointTypeResource]
		if count > 0 {
			id := vpcID
			usages = append(usages, QuotaUsage{
				Name:         resourceVpcEndpointsPerVpcName,
				ResourceName: &id,
				Description:  resourceVpcEndpointsPerVpcDesc,
				Usage:        float64(count),
			})
		}
	}

	return usages, nil
}

// ServiceNetworkVpcEndpointsPerVpcUsageCheck implements the UsageCheck interface
// for ServiceNetwork VPC endpoints per VPC (quota L-3B4E38D2).
type ServiceNetworkVpcEndpointsPerVpcUsageCheck struct {
	client ec2iface.EC2API
}

// Usage returns the usage of ServiceNetwork VPC endpoints per VPC.
func (c *ServiceNetworkVpcEndpointsPerVpcUsageCheck) Usage() ([]QuotaUsage, error) {
	counts, err := vpcEndpointsByVpcAndType(c.client)
	if err != nil {
		return nil, err
	}

	var usages []QuotaUsage
	for vpcID, typeCounts := range counts {
		count := typeCounts[vpcEndpointTypeServiceNetwork]
		if count > 0 {
			id := vpcID
			usages = append(usages, QuotaUsage{
				Name:         serviceNetworkVpcEndpointsPerVpcName,
				ResourceName: &id,
				Description:  serviceNetworkVpcEndpointsPerVpcDesc,
				Usage:        float64(count),
			})
		}
	}

	return usages, nil
}
