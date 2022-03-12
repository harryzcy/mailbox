package storage

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	tableName = os.Getenv("DYNAMODB_TABLE")
)

type dynamodbStorage struct{}

var DynamoDB dynamodbStorage

// StoreInDynamoDB stores data in DynamoDB
func (s dynamodbStorage) Store(ctx context.Context, cfg aws.Config, item map[string]types.AttributeValue) error {
	svc := dynamodb.NewFromConfig(cfg)

	resp, err := svc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	if err != nil {
		return err
	}

	log.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)

	return nil
}
