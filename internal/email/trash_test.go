package email

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
				return mockUpdateItemAPI(func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					assert.Len(t, params.Key, 1)
					assert.IsType(t, params.Key["MessageID"], &types.AttributeValueMemberS{})
					assert.Equal(t,
						params.Key["MessageID"].(*types.AttributeValueMemberS).Value,
						"exampleMessageID",
					)

					assert.Contains(t, *params.UpdateExpression, "=")
					updateExpressionParts := strings.Split(*params.UpdateExpression, "=")
					assert.Equal(t, "SET TrashedTime", strings.TrimSpace(updateExpressionParts[0]))
					assert.Contains(t, params.ExpressionAttributeValues, strings.TrimSpace(updateExpressionParts[1]))

					assert.Equal(t, "attribute_not_exists(TrashedTime) AND NOT begins_with(TypeYearMonth, :v_type)",
						*params.ConditionExpression)

					return &dynamodb.UpdateItemOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, &types.ConditionalCheckFailedException{}
				})
			},
			messageID:   "",
			expectedErr: ErrAlreadyTrashed,
		},
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, ErrNotFound
				})
			},
			messageID:   "",
			expectedErr: ErrNotFound,
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
