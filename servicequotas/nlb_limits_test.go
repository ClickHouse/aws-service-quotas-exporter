package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
)

func TestNLBsPerRegionUsageWithError(t *testing.T) {
	mockClient := &mockELBV2Client{
		err:                           errors.New("some err"),
		DescribeLoadBalancersResponse: nil,
	}

	check := NLBsPerRegionUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestNLBsPerRegionUsage(t *testing.T) {
	testCases := []struct {
		name          string
		loadBalancers []types.LoadBalancer
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoLoadBalancers",
			loadBalancers: []types.LoadBalancer{},
			expectedUsage: []QuotaUsage{
				{
					Name:        nlbsPerRegionName,
					Description: nlbsPerRegionDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMixedTypes",
			loadBalancers: []types.LoadBalancer{
				{Type: types.LoadBalancerTypeEnumNetwork},
				{Type: types.LoadBalancerTypeEnumNetwork},
				{Type: types.LoadBalancerTypeEnumApplication},
				{Type: types.LoadBalancerTypeEnum("gateway")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        nlbsPerRegionName,
					Description: nlbsPerRegionDesc,
					Usage:       2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockELBV2Client{
				err: nil,
				DescribeLoadBalancersResponse: &elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: tc.loadBalancers,
				},
			}

			check := NLBsPerRegionUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

// nlbNetworkInterface returns a network interface whose first address
// is its primary one
func nlbNetworkInterface(description, zone string, ips ...string) ec2types.NetworkInterface {
	eni := ec2types.NetworkInterface{Description: aws.String(description)}
	if zone != "" {
		eni.AvailabilityZone = aws.String(zone)
	}
	for i, ip := range ips {
		eni.PrivateIpAddresses = append(eni.PrivateIpAddresses, ec2types.NetworkInterfacePrivateIpAddress{
			PrivateIpAddress: aws.String(ip),
			Primary:          aws.Bool(i == 0),
		})
	}
	return eni
}

func nlbIPUsage(loadBalancer, zone, ip, primary string) QuotaUsage {
	return QuotaUsage{
		Name:         ipsPerNLBAZName,
		ResourceName: aws.String(loadBalancer),
		Description:  ipsPerNLBAZDesc,
		Usage:        1,
		Quota:        8,
		Labels:       map[string]string{"availability_zone": zone, "ip": ip, "primary": primary},
	}
}

func TestIPsPerNLBAvailabilityZoneUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                               errors.New("some err"),
		DescribeNetworkInterfacesResponse: nil,
	}

	check := IPsPerNLBAvailabilityZoneUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestIPsPerNLBAvailabilityZoneUsage(t *testing.T) {
	testCases := []struct {
		name              string
		networkInterfaces []ec2types.NetworkInterface
		expectedUsage     []QuotaUsage
	}{
		{
			name:              "WithNoNetworkInterfaces",
			networkInterfaces: []ec2types.NetworkInterface{},
			expectedUsage:     []QuotaUsage{},
		},
		{
			name: "WithPrimaryAndSecondaryIPs",
			networkInterfaces: []ec2types.NetworkInterface{
				nlbNetworkInterface("ELB net/ingress/aaa", "us-west-2a", "10.0.1.23", "10.0.1.6"),
				nlbNetworkInterface("ELB net/ingress/aaa", "us-west-2b", "10.0.2.112"),
				nlbNetworkInterface("ELB net/privatelink-0/bbb", "us-west-2a", "10.0.1.37"),
			},
			expectedUsage: []QuotaUsage{
				nlbIPUsage("net/ingress/aaa", "us-west-2a", "10.0.1.23", "true"),
				nlbIPUsage("net/ingress/aaa", "us-west-2a", "10.0.1.6", "false"),
				nlbIPUsage("net/ingress/aaa", "us-west-2b", "10.0.2.112", "true"),
				nlbIPUsage("net/privatelink-0/bbb", "us-west-2a", "10.0.1.37", "true"),
			},
		},
		{
			name: "WithOtherNetworkInterfaces",
			networkInterfaces: []ec2types.NetworkInterface{
				nlbNetworkInterface("ELB app/some-alb/ccc", "us-west-2a", "10.0.1.50"),
				nlbNetworkInterface("Interface for NAT Gateway nat-0123", "us-west-2a", "10.0.1.51"),
				nlbNetworkInterface("ELB net/ingress/aaa", "", "10.0.1.52"),
				{PrivateIpAddresses: []ec2types.NetworkInterfacePrivateIpAddress{{PrivateIpAddress: aws.String("10.0.1.53")}}},
				{
					Description:        aws.String("ELB net/ingress/aaa"),
					AvailabilityZone:   aws.String("us-west-2a"),
					PrivateIpAddresses: []ec2types.NetworkInterfacePrivateIpAddress{{}},
				},
			},
			expectedUsage: []QuotaUsage{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeNetworkInterfacesResponse: &ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: tc.networkInterfaces,
				},
			}

			check := IPsPerNLBAvailabilityZoneUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
			assert.Equal(t, []ec2types.Filter{
				{Name: aws.String("interface-type"), Values: []string{"network_load_balancer"}},
			}, mockClient.NetworkInterfacesFilters)
		})
	}
}
