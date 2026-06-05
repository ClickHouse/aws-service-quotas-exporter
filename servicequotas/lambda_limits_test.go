package servicequotas

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/assert"
)

type mockedLambdaClient struct {
	Output *lambda.GetAccountSettingsOutput
	Err    error
}

func (m *mockedLambdaClient) GetAccountSettings(_ context.Context, _ *lambda.GetAccountSettingsInput, _ ...func(*lambda.Options)) (*lambda.GetAccountSettingsOutput, error) {
	return m.Output, m.Err
}

func TestLambdaConcurrentExecutionsLimitCheck_Usage(t *testing.T) {
	tests := []struct {
		name          string
		client        lambdaAPI
		output        *lambda.GetAccountSettingsOutput
		err           error
		expectedUsage []QuotaUsage
		expectedErr   error
	}{
		{
			name: "success",
			client: &mockedLambdaClient{
				Output: &lambda.GetAccountSettingsOutput{
					AccountLimit: &types.AccountLimit{
						ConcurrentExecutions: 100,
						CodeSizeUnzipped:     1000000,
						CodeSizeZipped:       500000,
					},
					AccountUsage: &types.AccountUsage{
						FunctionCount: 50,
						TotalCodeSize: 500000,
					},
				},
				Err: nil,
			},
			expectedUsage: []QuotaUsage{
				{
					Name:        "lambda_concurrent_executions_limit",
					Description: "Measures the maximum number of concurrent executions allowed for an AWS Lambda function.",
					Quota:       100,
					Usage:       50,
				},
				{
					Name:        "lambda_code_size_unzipped_limit_bytes",
					Description: "Measures the maximum size limit (in bytes) for the unzipped AWS Lambda function code.",
					Quota:       1000000,
					Usage:       500000,
				},
			},
			expectedErr: nil,
		},
		{
			name: "error",
			client: &mockedLambdaClient{
				Output: nil,
				Err:    errors.New("some error occurred"),
			},
			expectedUsage: nil,
			expectedErr:   errors.New("some error occurred"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &LambdaConcurrentExecutionsLimitCheck{
				client: test.client,
			}

			usages, err := c.Usage()

			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedUsage, usages)
		})
	}
}
