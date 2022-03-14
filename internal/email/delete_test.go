package email

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockDeleteItemAPI struct {
	mockDeleteItem   func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	mockDeleteObject func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func (m mockDeleteItemAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.mockDeleteItem(ctx, params, optFns...)
}

func (m mockDeleteItemAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.mockDeleteObject(ctx, params, optFns...)
}

func TestDelete(t *testing.T) {
	tableName = "table-for-delete"
	tests := []struct {
		client      func(t *testing.T) DeleteItemAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						t.Helper()

						assert.Equal(t, tableName, *params.TableName)
						assert.Len(t, params.Key, 1)
						assert.IsType(t, params.Key["MessageID"], &types.AttributeValueMemberS{})
						assert.Equal(t,
							params.Key["MessageID"].(*types.AttributeValueMemberS).Value,
							"exampleMessageID",
						)
						assert.Equal(t, "attribute_exists(TrashedTime)", *params.ConditionExpression)

						return &dynamodb.DeleteItemOutput{}, nil
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) DeleteItemAPI {
				return mockDeleteItemAPI{
					mockDeleteItem: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
						return &dynamodb.DeleteItemOutput{}, errors.Wrap(&types.ConditionalCheckFailedException{}, "")
					},
					mockDeleteObject: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return &s3.DeleteObjectOutput{}, nil
					},
				}
			},
			expectedErr: ErrNotTrashed,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := Delete(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
