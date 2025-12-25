package email

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/stretchr/testify/assert"
)

type mockQueryAPI func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)

func (m mockQueryAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m(ctx, params, optFns...)
}

func TestByYearMonth(t *testing.T) {
	env.TableName = "list-by-year-month-table-name"
	env.GsiIndexName = "gsi-index-name"
	tests := []struct {
		client              func(t *testing.T) platform.QueryAPI
		unmarshalListOfMaps func(l []map[string]dynamodbTypes.AttributeValue, out interface{}) error
		input               listQueryInput
		expected            listQueryResult
		expectedErr         error
	}{
		{
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					t.Helper()

					assert.Equal(t, env.TableName, *params.TableName)
					assert.Equal(t, env.GsiIndexName, *params.IndexName)
					assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
						"foo": &dynamodbTypes.AttributeValueMemberS{Value: "bar"},
					}, params.ExclusiveStartKey)
					assert.Equal(t, "#tym = :val", *params.KeyConditionExpression)
					assert.Equal(t, map[string]dynamodbTypes.AttributeValue{
						":val": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
					}, params.ExpressionAttributeValues)
					assert.Equal(t, map[string]string{
						"#tym": "TypeYearMonth",
					}, params.ExpressionAttributeNames)

					assert.False(t, *params.ScanIndexForward)

					assert.Equal(t, *params.FilterExpression, "attribute_not_exists(TrashedTime)")
					assert.Equal(t, *params.Limit, int32(10))

					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
					}, nil
				})
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
				showTrash: "exclude",
				pageSize:  10,
				lastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
					"foo": &dynamodbTypes.AttributeValueMemberS{Value: "bar"},
				},
			},
			expected: listQueryResult{
				items: []Item{
					{
						TimeIndex: TimeIndex{
							MessageID:    "exampleMessageID",
							Type:         "inbox",
							TimeReceived: "2022-03-12T01:01:01Z",
						},
						Unread: new(bool),
					},
				},
			},
		},
		{
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					assert.Equal(t, *params.FilterExpression, "attribute_exists(TrashedTime)")

					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
					}, nil
				})
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
				showTrash: "only",
			},
			expected: listQueryResult{
				items: []Item{
					{
						TimeIndex: TimeIndex{
							MessageID:    "exampleMessageID",
							Type:         "inbox",
							TimeReceived: "2022-03-12T01:01:01Z",
						},
						Unread: new(bool),
					},
				},
			},
		},
		{
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {

					assert.Nil(t, params.FilterExpression)

					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
					}, nil
				})
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
				showTrash: "include",
			},
			expected: listQueryResult{
				items: []Item{
					{
						TimeIndex: TimeIndex{
							MessageID:    "exampleMessageID",
							Type:         "inbox",
							TimeReceived: "2022-03-12T01:01:01Z",
						},
						Unread: new(bool),
					},
				},
			},
		},
		{
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Count: 0,
						Items: []map[string]dynamodbTypes.AttributeValue{},
					}, nil
				})
			},
			unmarshalListOfMaps: func(_ []map[string]dynamodbTypes.AttributeValue, _ interface{}) error {
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
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Count: 1,
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "invalid"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
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
		{
			client: func(t *testing.T) platform.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{}, errors.New("error")
				})
			},
			input: listQueryInput{
				emailType: "inbox",
				year:      "2022",
				month:     "03",
			},
			expectedErr: errors.New("error"),
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
