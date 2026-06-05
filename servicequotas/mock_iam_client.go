package servicequotas

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type mockIAMClient struct {
	err                    error
	ListRolesResponse      *iam.ListRolesOutput
	ListPoliciesResponse   *iam.ListPoliciesOutput
	ListPoliciesScopeInput types.PolicyScopeType
}

func (m *mockIAMClient) ListRoles(_ context.Context, _ *iam.ListRolesInput, _ ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.ListRolesResponse == nil {
		return &iam.ListRolesOutput{}, nil
	}
	return m.ListRolesResponse, nil
}

func (m *mockIAMClient) ListPolicies(_ context.Context, input *iam.ListPoliciesInput, _ ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	m.ListPoliciesScopeInput = input.Scope
	if m.err != nil {
		return nil, m.err
	}
	if m.ListPoliciesResponse == nil {
		return &iam.ListPoliciesOutput{}, nil
	}
	return m.ListPoliciesResponse, nil
}
