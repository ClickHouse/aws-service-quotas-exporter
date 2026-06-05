package servicequotas

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsservicequotas "github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/stretchr/testify/assert"
)

type mockServiceQuotasClient struct {
	err                       error
	serviceName               string
	ListServiceQuotasResponse *awsservicequotas.ListServiceQuotasOutput
	timesCalled               int
}

func (m *mockServiceQuotasClient) ListServiceQuotas(_ context.Context, input *awsservicequotas.ListServiceQuotasInput, _ ...func(*awsservicequotas.Options)) (*awsservicequotas.ListServiceQuotasOutput, error) {
	m.timesCalled++

	if m.err != nil {
		return nil, m.err
	}
	if *input.ServiceCode == m.serviceName {
		return m.ListServiceQuotasResponse, nil
	}
	return &awsservicequotas.ListServiceQuotasOutput{}, nil
}

type UsageCheckMock struct {
	err    error
	usages []QuotaUsage
}

func (m *UsageCheckMock) Usage() ([]QuotaUsage, error) {
	return m.usages, m.err
}

func TestQuotasAndUsageWithError(t *testing.T) {
	mockClient := &mockServiceQuotasClient{
		err:                       errors.New("some err"),
		ListServiceQuotasResponse: nil,
	}

	serviceQuotas := ServiceQuotas{quotasService: mockClient, services: []string{"ec2", "vpc"}}
	quotasAndUsage, err := serviceQuotas.QuotasAndUsage()

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFailedToListQuotas))
	assert.Nil(t, quotasAndUsage)
}

func TestQuotasAndUsageWithUsageError(t *testing.T) {
	mockClient := &mockServiceQuotasClient{
		serviceName: "ec2",
		ListServiceQuotasResponse: &awsservicequotas.ListServiceQuotasOutput{
			Quotas: []types.ServiceQuota{
				{
					QuotaCode: aws.String("L-1234"),
					Value:     aws.Float64(15),
				},
			},
		},
	}

	expectedErr := errors.New("some err")
	usageCheckMock := &UsageCheckMock{
		err:    expectedErr,
		usages: nil,
	}

	serviceQuotas := ServiceQuotas{
		quotasService: mockClient,
		serviceQuotasUsageChecks: map[string]UsageCheck{
			"L-1234": usageCheckMock,
		},
		services: []string{"ec2", "vpc"},
	}
	quotasAndUsage, err := serviceQuotas.QuotasAndUsage()

	assert.Equal(t, expectedErr, err)
	assert.Nil(t, quotasAndUsage)
}

func TestQuotasAndUsage(t *testing.T) {
	mockClient := &mockServiceQuotasClient{
		serviceName: "ec2",
		ListServiceQuotasResponse: &awsservicequotas.ListServiceQuotasOutput{
			Quotas: []types.ServiceQuota{
				{
					QuotaCode: aws.String("L-1234"),
					Value:     aws.Float64(15),
				},
				{
					QuotaCode: aws.String("L-NOTIMPLEMENTED"),
					Value:     aws.Float64(6),
				},
				{
					QuotaCode: aws.String("L-5678"),
					Value:     aws.Float64(2),
				},
			},
		},
	}

	firstUsageCheckMock := &UsageCheckMock{
		usages: []QuotaUsage{
			{
				Name:         "check_with_multiple_resources",
				ResourceName: aws.String("i-resource1"),
				Description:  "check with multiple resources",
				Usage:        10,
			},
			{
				Name:         "check_with_multiple_resources",
				ResourceName: aws.String("i-resource2"),
				Description:  "check with multiple resources",
				Usage:        3,
			},
		},
	}
	secondUsageCheckMock := &UsageCheckMock{
		usages: []QuotaUsage{
			{
				Name:        "some_check",
				Description: "some check",
				Usage:       1,
			},
		},
	}

	serviceQuotas := ServiceQuotas{
		quotasService: mockClient,
		serviceQuotasUsageChecks: map[string]UsageCheck{
			"L-1234": firstUsageCheckMock,
			"L-5678": secondUsageCheckMock,
		},
		services: []string{"ec2", "vpc"},
	}
	actualQuotasAndUsage, err := serviceQuotas.QuotasAndUsage()

	expectedQuotasAndUsage := []QuotaUsage{
		{
			Name:         "check_with_multiple_resources",
			ResourceName: aws.String("i-resource1"),
			Description:  "check with multiple resources",
			Usage:        10,
			Quota:        15,
		},
		{
			Name:         "check_with_multiple_resources",
			ResourceName: aws.String("i-resource2"),
			Description:  "check with multiple resources",
			Usage:        3,
			Quota:        15,
		},
		{
			Name:        "some_check",
			Description: "some check",
			Usage:       1,
			Quota:       2,
		},
	}

	expectedServiceQuotasAPICalls := 2

	assert.NoError(t, err)
	assert.Equal(t, expectedServiceQuotasAPICalls, mockClient.timesCalled)
	assert.Equal(t, expectedQuotasAndUsage, actualQuotasAndUsage)
}

func TestQuotasAndUsageChina(t *testing.T) {

	// This won't be called as aws china doesn't support service quotas currently.
	mockClientNotUsed := &mockServiceQuotasClient{
		serviceName: "ec2",
		ListServiceQuotasResponse: &awsservicequotas.ListServiceQuotasOutput{
			Quotas: []types.ServiceQuota{
				{
					QuotaCode: aws.String("L-1234"),
					Value:     aws.Float64(15),
				},
			},
		},
	}

	firstUsageCheckMockNotUsed := &UsageCheckMock{
		usages: []QuotaUsage{
			{
				Name:         "check_with_multiple_resources",
				ResourceName: aws.String("i-resource1"),
				Description:  "check with multiple resources",
				Usage:        10,
			},
		},
	}
	secondUsageCheckMockNotUsed := &UsageCheckMock{
		usages: []QuotaUsage{
			{
				Name:        "some_check",
				Description: "some check",
				Usage:       1,
			},
		},
	}

	serviceQuotas := ServiceQuotas{
		quotasService: mockClientNotUsed,
		isAwsChina:    true,
		serviceQuotasUsageChecks: map[string]UsageCheck{
			"L-1234": firstUsageCheckMockNotUsed,
			"L-5678": secondUsageCheckMockNotUsed,
		},
		otherUsageChecks: []UsageCheck{
			&UsageCheckMock{
				usages: []QuotaUsage{
					{
						Name:        "some_check",
						Description: "some check",
						Usage:       1,
						Quota:       2,
					},
				},
			},
		},
	}
	actualQuotasAndUsage, err := serviceQuotas.QuotasAndUsage()

	// Service quotas are currently not supported in AWS china
	expectedQuotasAndUsage := []QuotaUsage{
		{
			Name:        "some_check",
			Description: "some check",
			Usage:       1,
			Quota:       2,
		},
	}

	expectedServiceQuotasAPICalls := 0

	assert.NoError(t, err)
	assert.Equal(t, expectedServiceQuotasAPICalls, mockClientNotUsed.timesCalled)
	assert.Equal(t, expectedQuotasAndUsage, actualQuotasAndUsage)
}

func TestQuotaUsageIdentifier(t *testing.T) {
	testCases := []struct {
		name               string
		quotaName          string
		resourceName       *string
		expectedIdentifier string
	}{
		{
			name:               "WithResourceName",
			quotaName:          "thequota",
			resourceName:       aws.String("some-resource"),
			expectedIdentifier: "some-resource",
		},
		{
			name:               "WithoutResourceName",
			quotaName:          "somequota",
			resourceName:       nil,
			expectedIdentifier: "somequota",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			usage := QuotaUsage{
				Name:         tc.quotaName,
				ResourceName: tc.resourceName,
			}
			assert.Equal(t, tc.expectedIdentifier, usage.Identifier())
		})
	}
}

func TestNewServiceQuotasWithInvalidRegion(t *testing.T) {
	svcQuotas, err := NewServiceQuotas("asdasd", "someprofile")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRegion))
	assert.Nil(t, svcQuotas)
}

func TestIsValidRegion(t *testing.T) {
	cases := []struct {
		region          string
		wantValid       bool
		wantIsChina     bool
		wantPartitionID string
	}{
		{"us-east-1", true, false, awsPartitionID},
		{"eu-central-1", true, false, awsPartitionID},
		{"cn-north-1", true, true, awsCnPartitionID},
		{"us-gov-west-1", true, false, awsUsGovPartitionID},
		{"mx-central-1", true, false, awsPartitionID},
		{"xx-fakeregion-9", true, false, awsPartitionID},
		{"asdasd", false, false, ""},
		{"", false, false, ""},
		{"us-east", false, false, ""},
		{"US-EAST-1", false, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.region, func(t *testing.T) {
			gotValid, gotChina, gotPartition := isValidRegion(tc.region)
			assert.Equal(t, tc.wantValid, gotValid, "valid")
			assert.Equal(t, tc.wantIsChina, gotChina, "isChina")
			assert.Equal(t, tc.wantPartitionID, gotPartition, "partitionID")
		})
	}
}

func TestQuotasForGlobalServiceRoutesToGlobalClient(t *testing.T) {
	regionalClient := &mockServiceQuotasClient{
		serviceName: "iam",
		ListServiceQuotasResponse: &awsservicequotas.ListServiceQuotasOutput{
			Quotas: []types.ServiceQuota{
				{QuotaCode: aws.String("L-REGIONAL"), Value: aws.Float64(1)},
			},
		},
	}
	globalClient := &mockServiceQuotasClient{
		serviceName: "iam",
		ListServiceQuotasResponse: &awsservicequotas.ListServiceQuotasOutput{
			Quotas: []types.ServiceQuota{
				{QuotaCode: aws.String("L-GLOBAL"), Value: aws.Float64(2)},
			},
		},
	}

	usageCheck := &UsageCheckMock{
		usages: []QuotaUsage{
			{Name: "iam_check", Description: "iam check", Usage: 1},
		},
	}

	serviceQuotas := ServiceQuotas{
		quotasService:       regionalClient,
		globalQuotasService: globalClient,
		serviceQuotasUsageChecks: map[string]UsageCheck{
			"L-GLOBAL": usageCheck,
		},
		services: []string{"iam"},
	}
	actual, err := serviceQuotas.QuotasAndUsage()

	assert.NoError(t, err)
	assert.Equal(t, 0, regionalClient.timesCalled, "regional client must not be called for global services")
	assert.Equal(t, 1, globalClient.timesCalled, "global client must be called for global services")
	assert.Equal(t, []QuotaUsage{
		{Name: "iam_check", Description: "iam check", Usage: 1, Quota: 2},
	}, actual)
}
