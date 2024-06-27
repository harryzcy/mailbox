package integration

import (
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
)

var (
	client *dynamodb.Client
)

func TestMain(m *testing.M) {
	client = newLocalClient()
	createTableIfNotExists(client)
	deleteAllItems()

	m.Run()
}

func newLocalClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
				Source: "Hard-coded credentials; values are irrelevant for local DynamoDB",
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})
}

func tableExists(d *dynamodb.Client) bool {
	tables, err := d.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		log.Fatal("ListTables failed", err)
	}
	for _, n := range tables.TableNames {
		if n == env.TableName {
			return true
		}
	}
	return false
}

func createTableIfNotExists(d *dynamodb.Client) {
	if tableExists(d) {
		log.Printf("table=%v already exists\n", env.TableName)
		return
	}
	_, err := d.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []dynamodbTypes.AttributeDefinition{
			{
				AttributeName: aws.String("MessageID"),
				AttributeType: dynamodbTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("TypeYearMonth"),
				AttributeType: dynamodbTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("DateTime"),
				AttributeType: dynamodbTypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("OriginalMessageID"),
				AttributeType: dynamodbTypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dynamodbTypes.KeySchemaElement{
			{
				AttributeName: aws.String("MessageID"),
				KeyType:       dynamodbTypes.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []dynamodbTypes.GlobalSecondaryIndex{
			{
				IndexName: aws.String("TimeIndex"),
				KeySchema: []dynamodbTypes.KeySchemaElement{
					{
						AttributeName: aws.String("TypeYearMonth"),
						KeyType:       dynamodbTypes.KeyTypeHash,
					},
					{
						AttributeName: aws.String("DateTime"),
						KeyType:       dynamodbTypes.KeyTypeRange,
					},
				},
				Projection: &dynamodbTypes.Projection{
					ProjectionType: dynamodbTypes.ProjectionTypeInclude,
					NonKeyAttributes: []string{
						"Subject",
						"From",
						"To",
						"Unread",
						"TrashedTime",
					},
				},
			},
			{
				IndexName: aws.String("OriginalMessageIDIndex"),
				KeySchema: []dynamodbTypes.KeySchemaElement{
					{
						AttributeName: aws.String("OriginalMessageID"),
						KeyType:       dynamodbTypes.KeyTypeHash,
					},
				},
				Projection: &dynamodbTypes.Projection{
					ProjectionType: dynamodbTypes.ProjectionTypeKeysOnly,
				},
			},
		},
		TableName:   aws.String(env.TableName),
		BillingMode: dynamodbTypes.BillingModePayPerRequest,
	})
	if err != nil {
		log.Fatal("CreateTable failed", err)
	}
	log.Printf("created table=%v\n", env.TableName)
}

func deleteAllItems() {
	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(env.TableName),
	})
	if err != nil {
		log.Fatal("Scan failed", err)
	}
	for _, item := range resp.Items {
		_, err := client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
			TableName: aws.String(env.TableName),
			Key:       map[string]dynamodbTypes.AttributeValue{"MessageID": item["MessageID"]},
		})
		if err != nil {
			log.Fatal("DeleteItem failed", err)
		}
	}
}
