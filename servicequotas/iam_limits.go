package servicequotas

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

const (
	iamRolesPerAccountName = "iam_roles_per_account"
	iamRolesPerAccountDesc = "IAM roles per account"

	iamPoliciesPerAccountName = "iam_policies_per_account"
	iamPoliciesPerAccountDesc = "IAM customer managed policies per account"
)

// IAMRolesPerAccountUsageCheck implements the UsageCheck interface for
// IAM roles per account (quota L-FE177D64).
type IAMRolesPerAccountUsageCheck struct {
	client iamiface.IAMAPI
}

// Usage returns the number of IAM roles in the account.
func (c *IAMRolesPerAccountUsageCheck) Usage() ([]QuotaUsage, error) {
	numRoles := 0

	err := c.client.ListRolesPages(&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			if page != nil {
				numRoles += len(page.Roles)
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	return []QuotaUsage{
		{
			Name:        iamRolesPerAccountName,
			Description: iamRolesPerAccountDesc,
			Usage:       float64(numRoles),
		},
	}, nil
}

// IAMPoliciesPerAccountUsageCheck implements the UsageCheck interface for
// IAM customer managed policies per account (quota L-E95E4862).
type IAMPoliciesPerAccountUsageCheck struct {
	client iamiface.IAMAPI
}

// Usage returns the number of customer managed IAM policies in the account.
// AWS managed policies are excluded by scoping the request to Local.
func (c *IAMPoliciesPerAccountUsageCheck) Usage() ([]QuotaUsage, error) {
	numPolicies := 0

	params := &iam.ListPoliciesInput{Scope: aws.String(iam.PolicyScopeTypeLocal)}
	err := c.client.ListPoliciesPages(params,
		func(page *iam.ListPoliciesOutput, lastPage bool) bool {
			if page != nil {
				numPolicies += len(page.Policies)
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
	}

	return []QuotaUsage{
		{
			Name:        iamPoliciesPerAccountName,
			Description: iamPoliciesPerAccountDesc,
			Usage:       float64(numPolicies),
		},
	}, nil
}
