package thread

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/stretchr/testify/assert"
)

type mockUpdateItemAPI func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)

func (m mockUpdateItemAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m(ctx, params, optFns...)
}

func TestTrash(t *testing.T) {
	tests := []struct {
		client      func(t *testing.T) api.UpdateItemAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, params *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &dynamodbTypes.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value,
						"exampleThreadID",
					)

					assert.Contains(t, *params.UpdateExpression, "=")
					updateExpressionParts := strings.Split(*params.UpdateExpression, "=")
					assert.Equal(t, "SET TrashedTime", strings.TrimSpace(updateExpressionParts[0]))
					assert.Contains(t, params.ExpressionAttributeValues, strings.TrimSpace(updateExpressionParts[1]))

					assert.Equal(t, "attribute_not_exists(TrashedTime)",
						*params.ConditionExpression)

					return &dynamodb.UpdateItemOutput{}, nil
				})
			},
			messageID: "exampleThreadID",
		},
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, &dynamodbTypes.ConditionalCheckFailedException{}
				})
			},
			messageID:   "",
			expectedErr: &api.NotTrashedError{Type: "thread"},
		},
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, api.ErrNotFound
				})
			},
			messageID:   "",
			expectedErr: api.ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := Trash(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
