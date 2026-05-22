// Package servicequotas implements logic for retrieving quotas for specific
// AWS services
package servicequotas

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	awsservicequotas "github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/service/servicequotas/servicequotasiface"
	logging "github.com/sirupsen/logrus"
)

var awsRegionPattern = regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)

// Errors returned from this package
var (
	ErrInvalidRegion       = errors.New("invalid region")
	ErrFailedToListQuotas  = errors.New("failed to list quotas")
	ErrFailedToGetUsage    = errors.New("failed to get usage")
	ErrFailedToConvertCidr = errors.New("failed to convert CIDR block from string to int")
)

// globalServiceQuotasRegions maps an AWS partition ID to the region from which
// the Service Quotas API exposes quotas for globally-scoped services (e.g. IAM).
// Listing quotas for these services from any other region returns an empty result.
var globalServiceQuotasRegions = map[string]string{
	endpoints.AwsPartitionID:      "us-east-1",
	endpoints.AwsUsGovPartitionID: "us-gov-west-1",
	endpoints.AwsCnPartitionID:    "cn-north-1",
}

// globalServices is the set of AWS service codes whose quotas must be queried
// from the partition's global Service Quotas region rather than the caller's
// configured region.
var globalServices = map[string]struct{}{
	"iam": {},
}

func isGlobalService(service string) bool {
	_, ok := globalServices[service]
	return ok
}

func allServices(opts QuotasOptions) []string {
	services := []string{"ec2", "vpc"}
	if opts.EnableNLBsPerRegionCheck {
		services = append(services, "elasticloadbalancing")
	}
	if opts.EnableIAMRolesPerAccountCheck || opts.EnableIAMPoliciesPerAccountCheck {
		services = append(services, "iam")
	}
	return services
}

// UsageCheck is an interface for retrieving service quota usage
type UsageCheck interface {
	// Usage returns slice of QuotaUsage or an error
	Usage() ([]QuotaUsage, error)
}

// QuotasOptions configures optional usage checks
type QuotasOptions struct {
	// EnableVpcEndpointChecks enables VPC endpoint quota monitoring
	EnableVpcEndpointChecks bool
	// EnableVpcsPerRegionCheck enables VPCs per region quota monitoring
	EnableVpcsPerRegionCheck bool
	// EnableEIPsPerRegionCheck enables Elastic IPs per region quota monitoring
	EnableEIPsPerRegionCheck bool
	// EnableNLBsPerRegionCheck enables Network Load Balancers per region quota monitoring
	EnableNLBsPerRegionCheck bool
	// EnableIAMRolesPerAccountCheck enables IAM roles per account quota monitoring
	EnableIAMRolesPerAccountCheck bool
	// EnableIAMPoliciesPerAccountCheck enables IAM customer managed policies per account quota monitoring
	EnableIAMPoliciesPerAccountCheck bool
}

func newUsageChecks(opts QuotasOptions, c client.ConfigProvider, cfgs ...*aws.Config) (map[string]UsageCheck, []UsageCheck) {
	// all clients that will be used by the usage checks
	ec2Client := ec2.New(c, cfgs...)
	autoscalingClient := autoscaling.New(c, cfgs...)
	lambdaClient := lambda.New(c, cfgs...)
	elbv2Client := elbv2.New(c, cfgs...)
	iamClient := iam.New(c, cfgs...)

	serviceQuotasUsageChecks := map[string]UsageCheck{
		"L-0EA8095F": &RulesPerSecurityGroupUsageCheck{ec2Client},
		"L-2AFB9258": &SecurityGroupsPerENIUsageCheck{ec2Client},
		"L-E79EC296": &SecurityGroupsPerRegionUsageCheck{ec2Client},
		"L-34B43A08": &StandardSpotInstanceRequestsUsageCheck{ec2Client},
		"L-1216C47A": &RunningOnDemandStandardInstancesUsageCheck{ec2Client},
	}

	if opts.EnableVpcEndpointChecks {
		serviceQuotasUsageChecks["L-29B6F2EB"] = &InterfaceVpcEndpointsPerVpcUsageCheck{client: ec2Client}
		serviceQuotasUsageChecks["L-CA6CC422"] = &ResourceVpcEndpointsPerVpcUsageCheck{client: ec2Client}
		serviceQuotasUsageChecks["L-3B4E38D2"] = &ServiceNetworkVpcEndpointsPerVpcUsageCheck{client: ec2Client}
	}

	if opts.EnableVpcsPerRegionCheck {
		serviceQuotasUsageChecks["L-F678F1CE"] = &VpcsPerRegionUsageCheck{client: ec2Client}
	}

	if opts.EnableEIPsPerRegionCheck {
		serviceQuotasUsageChecks["L-0263D0A3"] = &EIPsPerRegionUsageCheck{client: ec2Client}
	}

	if opts.EnableNLBsPerRegionCheck {
		serviceQuotasUsageChecks["L-69A177A2"] = &NLBsPerRegionUsageCheck{client: elbv2Client}
	}

	if opts.EnableIAMRolesPerAccountCheck {
		serviceQuotasUsageChecks["L-FE177D64"] = &IAMRolesPerAccountUsageCheck{client: iamClient}
	}

	if opts.EnableIAMPoliciesPerAccountCheck {
		serviceQuotasUsageChecks["L-E95E4862"] = &IAMPoliciesPerAccountUsageCheck{client: iamClient}
	}

	otherUsageChecks := []UsageCheck{
		&AvailableIpsPerSubnetUsageCheck{ec2Client},
		&ASGUsageCheck{autoscalingClient},
		&LambdaConcurrentExecutionsLimitCheck{lambdaClient},
	}

	return serviceQuotasUsageChecks, otherUsageChecks
}

// QuotaUsage represents service quota usage
type QuotaUsage struct {
	// Name is the name of the quota (eg. spot_instance_requests)
	// or the name given to the piece of exported availibility
	// information (eg. available_IPs_per_subnet)
	Name string
	// ResourceName is the name of the resource in case the quota
	// is for multiple resources. As an example for "rules per
	// security group" the ResourceName will be the ARN of the
	// security group.
	ResourceName *string
	// Description is the name of the service quota (eg. "Inbound
	// or outbound rules per security group")
	Description string
	// Usage is the current service quota usage
	Usage float64
	// Quota is the current quota
	Quota float64

	// Tags are the metadata associated with the resource in form of key, value pairs
	Tags map[string]string
}

// Identifier for the service quota. Either the resource name in case
// the quota is for multiple resources or the name of the quota
func (q QuotaUsage) Identifier() string {
	if q.ResourceName != nil {
		return *q.ResourceName
	}
	return q.Name
}

// ServiceQuotas is an implementation for retrieving service quotas
// and their limits
type ServiceQuotas struct {
	session                  *session.Session
	region                   string
	isAwsChina               bool
	quotasService            servicequotasiface.ServiceQuotasAPI
	globalQuotasService      servicequotasiface.ServiceQuotasAPI
	serviceQuotasUsageChecks map[string]UsageCheck
	otherUsageChecks         []UsageCheck
	services                 []string
}

// QuotasInterface is an interface for retrieving AWS service
// quotas and usage
type QuotasInterface interface {
	QuotasAndUsage() ([]QuotaUsage, error)
}

// NewServiceQuotas creates a ServiceQuotas for `region` and `profile`
// or returns an error. Note that the ServiceQuotas will only return
// usage and quotas for the service quotas with implemented usage checks
func NewServiceQuotas(region, profile string, quotasOpts ...QuotasOptions) (QuotasInterface, error) {
	validRegion, isChina, partitionID := isValidRegion(region)
	if !validRegion {
		return nil, fmt.Errorf("%w: failed to create ServiceQuotas", ErrInvalidRegion)
	}

	opts := session.Options{}
	if profile != "" {
		opts = session.Options{Profile: profile, SharedConfigState: session.SharedConfigEnable}
	}

	awsSession, err := session.NewSessionWithOptions(opts)
	if err != nil {
		return nil, err
	}

	var qo QuotasOptions
	if len(quotasOpts) > 0 {
		qo = quotasOpts[0]
	}

	quotasService := awsservicequotas.New(awsSession, aws.NewConfig().WithRegion(region))
	globalQuotasService := quotasService
	if globalRegion, ok := globalServiceQuotasRegions[partitionID]; ok && globalRegion != region {
		globalQuotasService = awsservicequotas.New(awsSession, aws.NewConfig().WithRegion(globalRegion))
	}
	serviceQuotasChecks, otherChecks := newUsageChecks(qo, awsSession, aws.NewConfig().WithRegion(region))

	if isChina {
		logging.Warn("AWS china currently doesn't support service quotas, disabling...")
	}

	quotas := &ServiceQuotas{
		session:                  awsSession,
		region:                   region,
		quotasService:            quotasService,
		globalQuotasService:      globalQuotasService,
		serviceQuotasUsageChecks: serviceQuotasChecks,
		isAwsChina:               isChina,
		otherUsageChecks:         otherChecks,
		services:                 allServices(qo),
	}
	return quotas, nil
}

func isValidRegion(region string) (bool, bool, string) {
	for _, partition := range endpoints.DefaultPartitions() {
		_, ok := partition.Regions()[region]
		if ok {
			return true, partition.ID() == endpoints.AwsCnPartitionID, partition.ID()
		}
	}
	if !strings.HasPrefix(region, "us-gov-") && !strings.HasPrefix(region, "cn-") &&
		awsRegionPattern.MatchString(region) {
		return true, false, endpoints.AwsPartitionID
	}
	return false, false, ""
}

func (s *ServiceQuotas) quotasForService(service string) ([]QuotaUsage, error) {
	serviceQuotaUsages := []QuotaUsage{}
	var usageErr error

	quotasService := s.quotasService
	if isGlobalService(service) && s.globalQuotasService != nil {
		quotasService = s.globalQuotasService
	}

	params := &awsservicequotas.ListServiceQuotasInput{ServiceCode: aws.String(service)}
	err := quotasService.ListServiceQuotasPages(params,
		func(page *awsservicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page != nil {
				for _, quota := range page.Quotas {
					if check, ok := s.serviceQuotasUsageChecks[*quota.QuotaCode]; ok {
						quotaUsages, err := check.Usage()
						if err != nil {
							usageErr = err
							// stop paging when an error is encountered
							return true
						}

						for _, quotaUsage := range quotaUsages {
							quotaUsage.Quota = *quota.Value
							serviceQuotaUsages = append(serviceQuotaUsages, quotaUsage)
						}
					}
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToListQuotas, err)
	}

	if usageErr != nil {
		return nil, usageErr
	}

	return serviceQuotaUsages, nil
}

// QuotasAndUsage returns a slice of `QuotaUsage` or an error
func (s *ServiceQuotas) QuotasAndUsage() ([]QuotaUsage, error) {
	allQuotaUsages := []QuotaUsage{}

	if !s.isAwsChina {
		for _, service := range s.services {
			serviceQuotas, err := s.quotasForService(service)
			if err != nil {
				return nil, err
			}

			for _, quota := range serviceQuotas {
				allQuotaUsages = append(allQuotaUsages, quota)
			}
		}
	}

	for _, check := range s.otherUsageChecks {
		quotas, err := check.Usage()
		if err != nil {
			return nil, err
		}

		for _, quota := range quotas {
			allQuotaUsages = append(allQuotaUsages, quota)
		}
	}

	return allQuotaUsages, nil
}
