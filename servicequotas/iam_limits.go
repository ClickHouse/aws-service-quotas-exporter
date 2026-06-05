package servicequotas

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const (
	iamRolesPerAccountName = "iam_roles_per_account"
	iamRolesPerAccountDesc = "IAM roles per account"

	iamPoliciesPerAccountName = "iam_policies_per_account"
	iamPoliciesPerAccountDesc = "IAM customer managed policies per account"
)

// iamAPI is the subset of the IAM client used by the IAM usage checks.
type iamAPI interface {
	ListRoles(context.Context, *iam.ListRolesInput, ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	ListPolicies(context.Context, *iam.ListPoliciesInput, ...func(*iam.Options)) (*iam.ListPoliciesOutput, error)
}

// IAMRolesPerAccountUsageCheck implements the UsageCheck interface for
// IAM roles per account (quota L-FE177D64).
type IAMRolesPerAccountUsageCheck struct {
	client iamAPI
}

// Usage returns the number of IAM roles in the account.
func (c *IAMRolesPerAccountUsageCheck) Usage() ([]QuotaUsage, error) {
	numRoles := 0

	paginator := iam.NewListRolesPaginator(c.client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}
		numRoles += len(page.Roles)
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
	client iamAPI
}

// Usage returns the number of customer managed IAM policies in the account.
// AWS managed policies are excluded by scoping the request to Local.
func (c *IAMPoliciesPerAccountUsageCheck) Usage() ([]QuotaUsage, error) {
	numPolicies := 0

	params := &iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal}
	paginator := iam.NewListPoliciesPaginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToGetUsage, err)
		}
		numPolicies += len(page.Policies)
	}

	return []QuotaUsage{
		{
			Name:        iamPoliciesPerAccountName,
			Description: iamPoliciesPerAccountDesc,
			Usage:       float64(numPolicies),
		},
	}, nil
}
