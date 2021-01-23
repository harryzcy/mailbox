package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var tableName = os.Getenv("DYNAMODB_TABLE")

func storeInDynamoDB(cfg aws.Config, item map[string]types.AttributeValue) error {
	svc := dynamodb.NewFromConfig(cfg)

	resp, err := svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	if err != nil {
		return err
	}

	// log.Printf("consumed %d DynamoDB capacity units", resp.ConsumedCapacity.CapacityUnits)
	log.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)

	return nil
}
