package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
)

// DynamoDBStorage is an interface that defines required DynamoDB functions
type DynamoDBStorage interface {
	Store(ctx context.Context, api DynamoDBPutItemAPI, item map[string]types.AttributeValue) error
}

type dynamodbStorage struct{}

// DynamoDB is the default implementation of DynamoDB related functions
var DynamoDB DynamoDBStorage = dynamodbStorage{}

// DynamoDBPutItemAPI defines set of API required by Store functions
type DynamoDBPutItemAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// StoreInDynamoDB stores data in DynamoDB
func (s dynamodbStorage) Store(ctx context.Context, api DynamoDBPutItemAPI, item map[string]types.AttributeValue) error {
	_, err := api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &env.TableName,
		Item:      item,
	})
	if err != nil {
		return err
	}

	return nil
}
