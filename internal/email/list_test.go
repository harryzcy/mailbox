package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type mockQueryAPI func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)

func (m mockQueryAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m(ctx, params, optFns...)
}
func TestList(t *testing.T) {
	tableName = "list-table-name"
	gsiIndexName = "gsi-index-name"
	tests := []struct {
		client      func(t *testing.T) QueryAPI
		year        string
		month       string
		expected    *ListResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) QueryAPI {
				return mockQueryAPI(func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					t.Helper()

					assert.Equal(t, tableName, *params.TableName)
					assert.Equal(t, gsiIndexName, *params.IndexName)
					assert.Equal(t, "#tym = :val", *params.KeyConditionExpression)
					assert.Equal(t, map[string]types.AttributeValue{
						":val": &types.AttributeValueMemberS{Value: "inbox#2022-03"},
					}, params.ExpressionAttributeValues)
					assert.Equal(t, map[string]string{
						"#tym": "TypeYearMonth",
					}, params.ExpressionAttributeNames)

					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]types.AttributeValue{
							{
								"MessageID":     &types.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &types.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
					}, nil
				})
			},
			year:  "2022",
			month: "3",
			expected: &ListResult{
				Count: 1,
				Items: []TimeIndex{
					{
						MessageID:    "exampleMessageID",
						Type:         "inbox",
						TimeReceived: "2022-03-12T01:01:01Z",
					},
				},
			},
		},
		{
			client: func(t *testing.T) QueryAPI {
				return mockQueryAPI(func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					t.Helper()
					t.Error("query should not run")
					return &dynamodb.QueryOutput{}, nil
				})
			},
			year:        "999",
			month:       "1",
			expectedErr: ErrInvalidInput,
		},
		{
			client: func(t *testing.T) QueryAPI {
				return mockQueryAPI(func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					t.Helper()
					t.Error("query should not run")
					return &dynamodb.QueryOutput{}, nil
				})
			},
			year:        "2021",
			month:       "100",
			expectedErr: ErrInvalidInput,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			result, err := List(ctx, test.client(t), test.year, test.month)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
