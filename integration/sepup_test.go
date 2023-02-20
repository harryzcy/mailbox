package integration

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	tableName = os.Getenv("DYNAMODB_TABLE")

	client *dynamodb.Client
)

func TestMain(m *testing.M) {
	client = newLocalClient()
	createTableIfNotExists(client)

	m.Run()
}

func newLocalClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           "http://localhost:8000",
				SigningRegion: region,
			}, nil
		})),
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

	return dynamodb.NewFromConfig(cfg)
}

func tableExists(d *dynamodb.Client) bool {
	tables, err := d.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		log.Fatal("ListTables failed", err)
	}
	for _, n := range tables.TableNames {
		if n == tableName {
			return true
		}
	}
	return false
}

func createTableIfNotExists(d *dynamodb.Client) {
	if tableExists(d) {
		log.Printf("table=%v already exists\n", tableName)
		return
	}
	_, err := d.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("MessageID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("TypeYearMonth"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("DateTime"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("OriginalMessageID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("MessageID"),
				KeyType:       types.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("TimeIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("TypeYearMonth"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("DateTime"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeInclude,
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
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("OriginalMessageID"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeKeysOnly,
				},
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		log.Fatal("CreateTable failed", err)
	}
	log.Printf("created table=%v\n", tableName)
}

func deleteAllItems() {
	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Fatal("Scan failed", err)
	}
	for _, item := range resp.Items {
		_, err := client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
			TableName: aws.String(tableName),
			Key:       map[string]types.AttributeValue{"MessageID": item["MessageID"]},
		})
		if err != nil {
			log.Fatal("DeleteItem failed", err)
		}
	}
}
