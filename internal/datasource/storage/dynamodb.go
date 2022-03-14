package storage

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	tableName = os.Getenv("DYNAMODB_TABLE")
)

type dynamodbStorage struct{}

var DynamoDB dynamodbStorage

type DynamoDBPutItemAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// StoreInDynamoDB stores data in DynamoDB
func (s dynamodbStorage) Store(ctx context.Context, api DynamoDBPutItemAPI, item map[string]types.AttributeValue) error {
	resp, err := api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	if err != nil {
		return err
	}

	log.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)

	return nil
}
