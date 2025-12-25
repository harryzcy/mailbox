package thread

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
)

func Trash(ctx context.Context, client platform.UpdateItemAPI, threadID string) error {
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: threadID},
		},
		UpdateExpression:    aws.String("SET TrashedTime = :val1"),
		ConditionExpression: aws.String("attribute_not_exists(TrashedTime)"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":val1": &dynamodbTypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return &platform.NotTrashedError{Type: "thread"}
		}
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}

		return err
	}

	fmt.Println("trash thread finished successfully")
	return nil
}
