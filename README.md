# AWS Service Quotas Exporter
The aws-service-quotas-exporter exports [AWS service quotas][1] and
usage as [Prometheus][2] metrics. This exporter only uses the service
quotas API and has custom implementation for each usage metric.
That makes it suitable for AWS accounts that do not have [Business or Enterprise
support plan][3], required by the [AWS Support API][4] (AWS
Trusted Advisor). This exporter also provides some metrics that are
not available via the AWS Trusted Advisor, such as "rules per security
group" and "spot instance requests". Other metrics exported through other AWS APIs
can also be integrated with minimal effort, an example of such is "available IPs
per subnet" as seen in //service_quotas/ec2_limits.go.

# Metrics

There are 7 metrics exposed by default, with additional optional metrics available via flags:

1. Rules per security group
```
aws_inbound_rules_per_security_group_limit_total{region="eu-west-1",resource="sg-0000000000000"} 200
aws_inbound_rules_per_security_group_used_total{region="eu-west-1",resource="sg-0000000000000"} 198
aws_outbound_rules_per_security_group_limit_total{region="eu-west-1",resource="sg-00000000000000"} 200
aws_outbound_rules_per_security_group_used_total{region="eu-west-1",resource="sg-00000000000000"} 7
```

2. Security groups per network interface
```
aws_security_groups_per_network_interface_limit_total{region="eu-west-1",resource="eni-00000000000"} 5
aws_security_groups_per_network_interface_used_total{region="eu-west-1",resource="eni-00000000000"} 1
```

3. Security groups per region
```
aws_security_groups_per_region_limit_total{region="eu-west-1",resource="security_groups_per_region"} 2500
aws_security_groups_per_region_used_total{region="eu-west-1",resource="security_groups_per_region"} 108
```

4. Spot instance requests
```
aws_spot_instance_requests_limit_total{region="eu-west-1",resource="spot_instance_requests"} 640
aws_spot_instance_requests_used_total{region="eu-west-1",resource="spot_instance_requests"} 472
```

5. On-demand instance requests
```
aws_ondemand_instance_requests_limit_total{region="eu-west-1",resource="ondemand_instance_requests"} 9088
aws_ondemand_instance_requests_used_total{region="eu-west-1",resource="ondemand_instance_requests"} 440
```

6. Available IPs per subnet
```
aws_available_ips_per_subnet_limit_total{region="eu-west-1",resource="subnet-do93c3jpg5oe4txjn"} 8192
aws_available_ips_per_subnet_used_total{region="eu-west-1",resource="subnet-do93c3jpg5oe4txjn"} 7959
```

7. VMs per AutoScalingGroup - useful to get alerts if the max number of instances for an ASG has been reached
```
aws_instances_per_asg_limit_total{region="eu-west-1",resource="asg"} 5
aws_instances_per_asg_used_total{region="eu-west-1",resource="asg"} 10
```

## Optional metrics (--enable-vpc-endpoints)

8. Interface VPC endpoints per VPC (includes Interface + GatewayLoadBalancer types, quota L-29B6F2EB)
```
aws_interface_vpc_endpoints_per_vpc_limit_total{region="us-east-1",resource="vpc-00000000000"} 500
aws_interface_vpc_endpoints_per_vpc_used_total{region="us-east-1",resource="vpc-00000000000"} 157
```

9. Resource VPC endpoints per VPC (quota L-CA6CC422)
```
aws_resource_vpc_endpoints_per_vpc_limit_total{region="us-east-1",resource="vpc-00000000000"} 500
aws_resource_vpc_endpoints_per_vpc_used_total{region="us-east-1",resource="vpc-00000000000"} 142
```

10. ServiceNetwork VPC endpoints per VPC (quota L-3B4E38D2)
```
aws_service_network_vpc_endpoints_per_vpc_limit_total{region="us-east-1",resource="vpc-00000000000"} 500
aws_service_network_vpc_endpoints_per_vpc_used_total{region="us-east-1",resource="vpc-00000000000"} 0
```

11. VPCs per region (quota L-F678F1CE, requires `--enable-vpcs-per-region`)
```
aws_vpcs_per_region_limit_total{region="us-east-1",resource="vpcs_per_region"} 5
aws_vpcs_per_region_used_total{region="us-east-1",resource="vpcs_per_region"} 3
```

12. EC2-VPC Elastic IPs per region (quota L-0263D0A3, requires `--enable-eips-per-region`)
```
aws_eips_per_region_limit_total{region="us-east-1",resource="eips_per_region"} 5
aws_eips_per_region_used_total{region="us-east-1",resource="eips_per_region"} 2
```

13. Network Load Balancers per region (quota L-69A177A2, requires `--enable-nlbs-per-region`)
```
aws_nlbs_per_region_limit_total{region="us-east-1",resource="nlbs_per_region"} 50
aws_nlbs_per_region_used_total{region="us-east-1",resource="nlbs_per_region"} 4
```

14. IAM roles per account (quota L-FE177D64, requires `--enable-iam-roles-per-account`). IAM is a global service, so enable this in only one region per account to avoid duplicate metrics. The exporter itself can run in any region — IAM quotas are fetched from the partition's global Service Quotas region (`us-east-1` for the standard AWS partition).
```
aws_iam_roles_per_account_limit_total{region="us-east-1",resource="iam_roles_per_account"} 1000
aws_iam_roles_per_account_used_total{region="us-east-1",resource="iam_roles_per_account"} 312
```

15. IAM customer managed policies per account (quota L-E95E4862, requires `--enable-iam-policies-per-account`). Same global-service caveat as above.
```
aws_iam_policies_per_account_limit_total{region="us-east-1",resource="iam_policies_per_account"} 1500
aws_iam_policies_per_account_used_total{region="us-east-1",resource="iam_policies_per_account"} 47
```

# IAM Permissions

The AWS Service Quotas requires permissions for the following actions
to be able to run:

 * `ec2:DescribeSecurityGroups`
 * `ec2:DescribeNetworkInterfaces`
 * `ec2:DescribeInstances`
 * `ec2:DescribeSubnets`
 * `ec2:DescribeVpcEndpoints` (only when `--enable-vpc-endpoints` is used)
 * `ec2:DescribeVpcs` (only when `--enable-vpcs-per-region` is used)
 * `ec2:DescribeAddresses` (only when `--enable-eips-per-region` is used)
 * `elasticloadbalancing:DescribeLoadBalancers` (only when `--enable-nlbs-per-region` is used)
 * `iam:ListRoles` (only when `--enable-iam-roles-per-account` is used)
 * `iam:ListPolicies` (only when `--enable-iam-policies-per-account` is used)
 * `servicequotas:ListServiceQuotas`
 * `autoscaling:DescribeAutoScalingGroups`

Example IAM policy
```
{
   "Version": "2012-10-17",
   "Statement": [{
      "Effect": "Allow",
      "Action": [
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DescribeInstances",
          "ec2:DescribeSubnets",
          "ec2:DescribeVpcEndpoints",
          "ec2:DescribeVpcs",
          "ec2:DescribeAddresses",
          "elasticloadbalancing:DescribeLoadBalancers",
          "iam:ListRoles",
          "iam:ListPolicies",
          "servicequotas:ListServiceQuotas",
          "autoscaling:DescribeAutoScalingGroups"
      ],
      "Resource": "*"
   }]
}
```

# Options

`go run cmd/main.go -- [OPTIONS]`
| Short Flag | Long Flag          | Env var                       | Description                                              |
|------------|--------------------|----------------------|-------------------------------------------------------------------|
| -p         | --port             | N/A         | Port on which to serve metrics                                             |
| -r         | --region           | AWS_REGION  | AWS region                                                                 |
| -f         | --profile          | AWS_PROFILE | Named AWS profile                                                          |
| N/A        | --include-aws-tag  | N/A         | The aws resource tags to include as labels for returned metrics            |
| N/A        | --enable-vpc-endpoints | N/A     | Enable VPC endpoint quota monitoring                                       |
| N/A        | --enable-vpcs-per-region | N/A   | Enable VPCs per region quota monitoring                                    |
| N/A        | --enable-eips-per-region | N/A   | Enable Elastic IPs per region quota monitoring                             |
| N/A        | --enable-nlbs-per-region | N/A   | Enable Network Load Balancers per region quota monitoring                  |
| N/A        | --enable-iam-roles-per-account | N/A | Enable IAM roles per account quota monitoring                          |
| N/A        | --enable-iam-policies-per-account | N/A | Enable IAM customer managed policies per account quota monitoring   |

# Building the exporter and running the exporter

## Building the binary
`go build -ldflags "-linkmode external -extldflags -static" -o aws-service-quotas-exporter cmd/main.go`

## Docker image
`docker build -t aws-quotas-exporter -f build/Dockerfile . --rm=false`

Docker images are also available at thoughtmachine/aws-service-quotas-exporter:<version> See https://hub.docker.com/r/thoughtmachine/aws-service-quotas-exporter

# Extending the exporter with additional metrics

### Implement the `QuotasInterface`.

Example
`service_quotas/<service_name>_limits.go`
```
const (
    myQuotaName        = "prometheus_valid_metric_name"  // Only [a-zA-Z0-9:_]
    myQuotaDescription = "my description"
)

type MyUsageCheck struct {
    client awsserviceiface.SERVICEAPI  // eg ec2iface.EC2API
}

func (c *MyUsageCheck) Usage() ([]QuotaUsage, error) {
    // ...client.GetRequiredInformation

    // In case we are retrieving usage for multiple resources:
    for _, resource := range {
        usage := QuotaUsage{
            Name:         myQuotaName,
            ResourceName: resource.Identifier,
            Description:  myQuotaDescription,
            Usage:        myUsage,
        }
        usages = append(usages, usage)
    }

    // For a single resource
    usages := []QuotaUsage{
        {
            Name:        myQuotaName,
            Description: myQuotaDescription,
            Usage:       myUsage,
        },
    }

    return usages, err
}
```

### Add the check to the `newUsageChecks` and make sure to pass the appropriate AWS client

If the check uses the Service Quotas API, then it needs to be added as part of
`serviceQuotasUsageChecks` with its service quota code (examples given in the
[using AWS CLI to manage service quota requests page][5]). Otherwise, the check can
just be added to `otherUsageChecks`.

`service_quotas/service_quotas.go`
```
func newUsageChecks(c client.ConfigProvider, cfgs ...*aws.Config) map[string]UsageCheck {
    myClient := someawsclient.New(c, cfgs)

    serviceQuotasUsageChecks := map[string]UsageCheck{
        //... other usage checks
        "L-SERVICE_QUOTAS_CODE": &MyUsageCheck{myClient},
    }

    otherUsageChecks := []UsageCheck{
        &MyOtherUsageCheck{ec2Client},
    }

    return serviceQuotasUsageChecks, otherUsageChecks
}
```

### Update this README with the required actions :) (See the IAM Permissions section)


[1]: https://docs.aws.amazon.com/general/latest/gr/aws_service_limits.html
[2]: https://prometheus.io/
[3]: https://aws.amazon.com/premiumsupport/plans/
[4]: https://docs.aws.amazon.com/awssupport/latest/APIReference/Welcome.html
[5]: https://aws.amazon.com/premiumsupport/knowledge-center/troubleshoot-service-quotas-cli-commands/
