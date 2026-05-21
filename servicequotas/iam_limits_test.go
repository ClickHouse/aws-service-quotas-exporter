package servicequotas

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestIAMRolesPerAccountUsageWithError(t *testing.T) {
	mockClient := &mockIAMClient{
		err:               errors.New("some err"),
		ListRolesResponse: nil,
	}

	check := IAMRolesPerAccountUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestIAMRolesPerAccountUsage(t *testing.T) {
	testCases := []struct {
		name          string
		roles         []*iam.Role
		expectedUsage []QuotaUsage
	}{
		{
			name:  "WithNoRoles",
			roles: []*iam.Role{},
			expectedUsage: []QuotaUsage{
				{
					Name:        iamRolesPerAccountName,
					Description: iamRolesPerAccountDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMultipleRoles",
			roles: []*iam.Role{
				{RoleName: aws.String("role-a")},
				{RoleName: aws.String("role-b")},
				{RoleName: aws.String("role-c")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        iamRolesPerAccountName,
					Description: iamRolesPerAccountDesc,
					Usage:       3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockIAMClient{
				err: nil,
				ListRolesResponse: &iam.ListRolesOutput{
					Roles: tc.roles,
				},
			}

			check := IAMRolesPerAccountUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
		})
	}
}

func TestIAMPoliciesPerAccountUsageWithError(t *testing.T) {
	mockClient := &mockIAMClient{
		err:                  errors.New("some err"),
		ListPoliciesResponse: nil,
	}

	check := IAMPoliciesPerAccountUsageCheck{mockClient}
	usage, err := check.Usage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToGetUsage))
	assert.Nil(t, usage)
}

func TestIAMPoliciesPerAccountUsage(t *testing.T) {
	testCases := []struct {
		name          string
		policies      []*iam.Policy
		expectedUsage []QuotaUsage
	}{
		{
			name:     "WithNoPolicies",
			policies: []*iam.Policy{},
			expectedUsage: []QuotaUsage{
				{
					Name:        iamPoliciesPerAccountName,
					Description: iamPoliciesPerAccountDesc,
					Usage:       0,
				},
			},
		},
		{
			name: "WithMultiplePolicies",
			policies: []*iam.Policy{
				{PolicyName: aws.String("policy-a")},
				{PolicyName: aws.String("policy-b")},
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        iamPoliciesPerAccountName,
					Description: iamPoliciesPerAccountDesc,
					Usage:       2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockIAMClient{
				err: nil,
				ListPoliciesResponse: &iam.ListPoliciesOutput{
					Policies: tc.policies,
				},
			}

			check := IAMPoliciesPerAccountUsageCheck{mockClient}
			usage, err := check.Usage()

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUsage, usage)
			assert.Equal(t, iam.PolicyScopeTypeLocal, aws.StringValue(mockClient.ListPoliciesScopeInput))
		})
	}
}
