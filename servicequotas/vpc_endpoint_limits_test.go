package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

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
		endpoints     []types.VpcEndpoint
		expectedUsage []QuotaUsage
	}{
		{
			name:          "WithNoEndpoints",
			endpoints:     []types.VpcEndpoint{},
			expectedUsage: nil,
		},
		{
			name: "WithInterfaceEndpoints",
			endpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("GatewayLoadBalancer"),
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
			endpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Gateway"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Resource"),
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
			endpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-aaa"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
				},
				{
					VpcId:           aws.String("vpc-bbb"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
				},
				{
					VpcId:           aws.String("vpc-bbb"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
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
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Resource"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Resource"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
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
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("ServiceNetwork"),
				},
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Interface"),
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
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcId:           aws.String("vpc-123"),
					VpcEndpointType: types.VpcEndpointType("Gateway"),
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
