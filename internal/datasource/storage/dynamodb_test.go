package storage

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type mockPutItemAPI func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)

func (m mockPutItemAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m(ctx, params, optFns...)
}

func TestDynamoDB_Store(t *testing.T) {
	tableName = "example-table"
	tests := []struct {
		client      func(t *testing.T, item map[string]types.AttributeValue) DynamoDBPutItemAPI
		item        map[string]types.AttributeValue
		expectedErr error
	}{
		{
			client: func(t *testing.T, item map[string]types.AttributeValue) DynamoDBPutItemAPI {
				return mockPutItemAPI(
					func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
						t.Helper()
						assert.Equal(t, tableName, *params.TableName)
						assert.Equal(t, item, params.Item)
						return &dynamodb.PutItemOutput{}, nil
					})
			},
			item: map[string]types.AttributeValue{"k": &types.AttributeValueMemberS{Value: "v"}},
		},
		{
			client: func(t *testing.T, item map[string]types.AttributeValue) DynamoDBPutItemAPI {
				return mockPutItemAPI(
					func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
						t.Helper()
						return &dynamodb.PutItemOutput{}, errors.New("some-error")
					})
			},
			expectedErr: errors.New("some-error"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := DynamoDB.Store(ctx, test.client(t, test.item), test.item)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
