package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/stretchr/testify/assert"
)

func TestUntrash(t *testing.T) {
	tests := []struct {
		client      func(t *testing.T) platform.UpdateItemAPI
		messageID   string
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
					assert.Equal(t, "REMOVE TrashedTime", *params.UpdateExpression)
					assert.Equal(t, "attribute_exists(TrashedTime) AND NOT begins_with(TypeYearMonth, :v_type)",
						*params.ConditionExpression)

					return &dynamodb.UpdateItemOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) platform.UpdateItemAPI {
				return mockUpdateItemAPI(func(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Helper()
					return &dynamodb.UpdateItemOutput{}, &dynamodbTypes.ConditionalCheckFailedException{}
				})
			},
			messageID:   "",
			expectedErr: &platform.NotTrashedError{Type: "email"},
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
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			err := Untrash(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
