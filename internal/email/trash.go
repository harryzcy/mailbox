package email

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Trash marks an email as trashed
func Trash(cfg aws.Config, messageID string) error {
	svc := dynamodb.NewFromConfig(cfg)

	_, err := svc.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"messageID": &types.AttributeValueMemberS{Value: messageID},
		},
		UpdateExpression: aws.String("SET trashedTime = :val1"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val1": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("trash method finished successfully")
	return nil
}
