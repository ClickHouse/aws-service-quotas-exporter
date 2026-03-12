package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

func (m *mockEC2Client) DescribeVpcEndpointsPages(input *ec2.DescribeVpcEndpointsInput, fn func(*ec2.DescribeVpcEndpointsOutput, bool) bool) error {
	fn(m.DescribeVpcEndpointsResponse, true)
	return m.err
}

func TestInterfaceVpcEndpointsPerVpcUsageWithError(t *testing.T) {
	mockClient := &mockEC2Client{
		err:                            errors.New("some err"),
		DescribeVpcEndpointsResponse: nil,
	}

	check := InterfaceVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestInterfaceVpcEndpointsPerVpcUsage(t *testing.T) {
	testCases := []struct {
		name          string
		endpoints     []*ec2.VpcEndpoint
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoEndpoints",
			endpoints:     []*ec2.VpcEndpoint{},
			expectedUsage: nil,
		},
		{
			name: "WithInterfaceEndpoints",
			endpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("GatewayLoadBalancer"),
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         interfaceVpcEndpointsPerVpcName,
					ResourceName: aws.String("vpc-123"),
					Description:  interfaceVpcEndpointsPerVpcDesc,
					Usage:        3,
				},
			},
		},
		{
			name: "WithMixedEndpointTypes",
			endpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Gateway"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Resource"),
				},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:         interfaceVpcEndpointsPerVpcName,
					ResourceName: aws.String("vpc-123"),
					Description:  interfaceVpcEndpointsPerVpcDesc,
					Usage:        1,
				},
			},
		},
		{
			name: "WithMultipleVPCs",
			endpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-aaa"),
					VpcEndpointType: aws.String("Interface"),
				},
				{
					VpcId:           aws.String("vpc-bbb"),
					VpcEndpointType: aws.String("Interface"),
				},
				{
					VpcId:           aws.String("vpc-bbb"),
					VpcEndpointType: aws.String("Interface"),
				},
			},
			expectedUsage: nil, // checked below with ElementsMatch due to map ordering
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockEC2Client{
				err: nil,
				DescribeVpcEndpointsResponse: &ec2.DescribeVpcEndpointsOutput{
					VpcEndpoints: tc.endpoints,
				},
			}

			check := InterfaceVpcEndpointsPerVpcUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)

			if tc.name == "WithMultipleVPCs" {
				assert.ElementsMatch(t, []QuotaUsage{
					{
						Name:         interfaceVpcEndpointsPerVpcName,
						ResourceName: aws.String("vpc-aaa"),
						Description:  interfaceVpcEndpointsPerVpcDesc,
						Usage:        1,
					},
					{
						Name:         interfaceVpcEndpointsPerVpcName,
						ResourceName: aws.String("vpc-bbb"),
						Description:  interfaceVpcEndpointsPerVpcDesc,
						Usage:        2,
					},
				}, usage)
			} else {
				assert.Equal(t, tc.expectedUsage, usage)
			}
		})
	}
}

func TestResourceVpcEndpointsPerVpcUsage(t *testing.T) {
	mockClient := &mockEC2Client{
		err: nil,
		DescribeVpcEndpointsResponse: &ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Resource"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Resource"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Interface"),
				},
			},
		},
	}

	check := ResourceVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.NoError(t, err)
	assert.Equal(t, []QuotaUsage{
		{
			Name:         resourceVpcEndpointsPerVpcName,
			ResourceName: aws.String("vpc-123"),
			Description:  resourceVpcEndpointsPerVpcDesc,
			Usage:        2,
		},
	}, usage)
}

func TestServiceNetworkVpcEndpointsPerVpcUsage(t *testing.T) {
	mockClient := &mockEC2Client{
		err: nil,
		DescribeVpcEndpointsResponse: &ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("ServiceNetwork"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Interface"),
				},
			},
		},
	}

	check := ServiceNetworkVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.NoError(t, err)
	assert.Equal(t, []QuotaUsage{
		{
			Name:         serviceNetworkVpcEndpointsPerVpcName,
			ResourceName: aws.String("vpc-123"),
			Description:  serviceNetworkVpcEndpointsPerVpcDesc,
			Usage:        1,
		},
	}, usage)
}

func TestVpcEndpointChecksWithNoMatchingType(t *testing.T) {
	mockClient := &mockEC2Client{
		err: nil,
		DescribeVpcEndpointsResponse: &ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []*ec2.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: aws.String("Gateway"),
				},
			},
		},
	}

	// Interface check should return no usages for Gateway-only endpoints
	interfaceCheck := InterfaceVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err := interfaceCheck.Usage()
	assert.NoError(t, err)
	assert.Nil(t, usage)

	// Resource check should return no usages for Gateway-only endpoints
	resourceCheck := ResourceVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err = resourceCheck.Usage()
	assert.NoError(t, err)
	assert.Nil(t, usage)

	// ServiceNetwork check should return no usages for Gateway-only endpoints
	snCheck := ServiceNetworkVpcEndpointsPerVpcUsageCheck{mockClient}
	usage, err = snCheck.Usage()
	assert.NoError(t, err)
	assert.Nil(t, usage)
}
