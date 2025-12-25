package email

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	tests := []struct {
		client      func(t *testing.T) platform.UpdateItemAPI
		messageID   string
		action      string
		expectedErr error
	}{
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
						"exampleMessageID",
					)

					assert.Equal(t, "REMOVE Unread", strings.TrimSpace(*params.UpdateExpression))
					assert.Equal(t, "attribute_exists(Unread) AND begins_with(TypeYearMonth, :v_type)",
						*params.ConditionExpression)

					return &dynamodb.UpdateItemOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
			action:    ActionRead,
		},
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
						"exampleMessageID",
					)

					assert.Contains(t, *params.UpdateExpression, "=")
					updateExpressionParts := strings.Split(*params.UpdateExpression, "=")
					assert.Equal(t, "SET Unread", strings.TrimSpace(updateExpressionParts[0]))
					assert.Contains(t, params.ExpressionAttributeValues, strings.TrimSpace(updateExpressionParts[1]))

					assert.Equal(t, "attribute_not_exists(Unread) AND begins_with(TypeYearMonth, :v_type)",
						*params.ConditionExpression)

					return &dynamodb.UpdateItemOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
			action:    ActionUnread,
		},
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, &dynamodbTypes.ConditionalCheckFailedException{}
				})
			},
			messageID:   "",
			expectedErr: platform.ErrReadActionFailed,
		},
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, platform.ErrNotFound
				})
			},
			messageID:   "",
			expectedErr: platform.ErrNotFound,
		},
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, &dynamodbTypes.ProvisionedThroughputExceededException{}
				})
			},
			expectedErr: platform.ErrTooManyRequests,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := Read(ctx, test.client(t), test.messageID, test.action)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
