package email

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/stretchr/testify/assert"
)

type mockQueryAPI func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)

func (m mockQueryAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m(ctx, params, optFns...)
}

func TestByYearMonth(t *testing.T) {
	tableName = "list-by-year-month-table-name"
	gsiIndexName = "gsi-index-name"
	tests := []struct {
		client              func(t *testing.T) QueryAPI
		unmarshalListOfMaps func(l []map[string]types.AttributeValue, out interface{}) error
		input               listQueryInput
		expected            listQueryResult
		expectedErr         error
	}{
		{
			client: func(t *testing.T) QueryAPI {
				return mockQueryAPI(func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					t.Helper()

					assert.Equal(t, tableName, *params.TableName)
					assert.Equal(t, gsiIndexName, *params.IndexName)
					assert.Equal(t, map[string]types.AttributeValue{
						"foo": &types.AttributeValueMemberS{Value: "bar"},
					}, params.ExclusiveStartKey)
					assert.Equal(t, "#tym = :val", *params.KeyConditionExpression)
					assert.Equal(t, map[string]types.AttributeValue{
						":val": &types.AttributeValueMemberS{Value: "inbox#2022-03"},
					}, params.ExpressionAttributeValues)
					assert.Equal(t, map[string]string{
						"#tym": "TypeYearMonth",
					}, params.ExpressionAttributeNames)

					assert.False(t, *params.ScanIndexForward)

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
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
				lastEvaluatedKey: map[string]types.AttributeValue{
					"foo": &types.AttributeValueMemberS{Value: "bar"},
				},
			},
			expected: listQueryResult{
				items: []TimeIndex{
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
					return &dynamodb.QueryOutput{
						Count: 0,
						Items: []map[string]types.AttributeValue{},
					}, nil
				})
			},
			unmarshalListOfMaps: func(l []map[string]types.AttributeValue, out interface{}) error {
				return errors.New("error")
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
			},
			expectedErr: errors.New("error"),
		},
		{
			client: func(t *testing.T) QueryAPI {
				return mockQueryAPI(func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]types.AttributeValue{
							{
								"MessageID":     &types.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &types.AttributeValueMemberS{Value: "invalid"},
								"DateTime":      &types.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
					}, nil
				})
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
			},
			expectedErr: format.ErrInvalidFormatForTypeYearMonth,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			originalUnmarshalListOfMaps := unmarshalListOfMaps
			if test.unmarshalListOfMaps != nil {
				unmarshalListOfMaps = test.unmarshalListOfMaps
			}

			result, err := listByYearMonth(ctx, test.client(t), test.input)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.expectedErr, err)

			unmarshalListOfMaps = originalUnmarshalListOfMaps
		})
	}
}
