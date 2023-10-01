package thread

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/stretchr/testify/assert"
)

func TestUntrash(t *testing.T) {
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
					assert.Equal(t, "REMOVE TrashedTime", *params.UpdateExpression)
					assert.Equal(t, "attribute_exists(TrashedTime)",
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
			expectedErr: email.ErrNotTrashed,
		},
		{
			client: func(t *testing.T) api.UpdateItemAPI {
				return mockUpdateItemAPI(func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, email.ErrNotFound
				})
			},
			messageID:   "",
			expectedErr: email.ErrNotFound,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := Untrash(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
