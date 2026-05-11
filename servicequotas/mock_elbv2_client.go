package servicequotas

import (
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

type mockELBV2Client struct {
	elbv2iface.ELBV2API

	err                           error
	DescribeLoadBalancersResponse *elbv2.DescribeLoadBalancersOutput
}
