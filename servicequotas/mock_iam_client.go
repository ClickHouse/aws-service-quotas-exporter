package servicequotas

import (
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type mockIAMClient struct {
	iamiface.IAMAPI

	err                    error
	ListRolesResponse      *iam.ListRolesOutput
	ListPoliciesResponse   *iam.ListPoliciesOutput
	ListPoliciesScopeInput *string
}

func (m *mockIAMClient) ListRolesPages(input *iam.ListRolesInput, fn func(*iam.ListRolesOutput, bool) bool) error {
	fn(m.ListRolesResponse, true)
	return m.err
}

func (m *mockIAMClient) ListPoliciesPages(input *iam.ListPoliciesInput, fn func(*iam.ListPoliciesOutput, bool) bool) error {
	m.ListPoliciesScopeInput = input.Scope
	fn(m.ListPoliciesResponse, true)
	return m.err
}
